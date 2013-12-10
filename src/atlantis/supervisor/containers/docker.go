package containers

import (
	"atlantis/crypto"
	"atlantis/supervisor/rpc/types"
	"fmt"
	"github.com/dotcloud/docker"
	dockercli "github.com/fsouza/go-dockerclient"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

var (
	dockerIdRegexp = regexp.MustCompile("^[A-Za-z0-9]+$")
	dockerLock     = sync.Mutex{}
	dockerClient   *dockercli.Client
)

type Container types.Container
type SSHCmd []string

func DockerInit() (err error) {
	dockerClient, err = dockercli.NewClient("unix:///var/run/docker.sock")
	return
}

func pretending() bool {
	return os.Getenv("SUPERVISOR_PRETEND") != ""
}

func (s SSHCmd) Execute() error {
	if pretending() {
		log.Printf("[pretend] ssh %s", strings.Join(s, " "))
		return nil
	}
	log.Printf("ssh %s", strings.Join(s, " "))
	cmd := exec.Command("ssh", s...)
	output, err := cmd.CombinedOutput()
	log.Printf("-> %s", output)
	if err != nil {
		log.Println("-> Error:", err)
	}
	return err
}

func RemoveExited() {
	if pretending() {
		return
	}
	dockerLock.Lock()
	defer dockerLock.Unlock()
	containers, err := dockerClient.ListContainers(dockercli.ListContainersOptions{All: true})
	if err != nil {
		log.Printf("[RemoveExited] could not list containers: %v", err)
		return
	}
	for _, cont := range containers {
		log.Printf("[RemoveExited] checking %s (%v) : %s", cont.ID, cont.Names, cont.Status)
		if !strings.HasPrefix(cont.Status, "Exit") {
			continue
		}
		log.Printf("[RemoveExited] remove %s (%v)", cont.ID, cont.Names)
		err := dockerClient.RemoveContainer(cont.ID)
		if err != nil {
			log.Printf("[RemoveExited] -> error: %v", err)
		} else {
			log.Printf("[RemoveExited] -> success", err)
		}
	}
}

func (c *Container) dockerCfgs(repo string) (*docker.Config, *docker.HostConfig) {
	// get env cfg
	envs := []string{
		"ATLANTIS=true",
		fmt.Sprintf("CONTAINER_ID=%s", c.Id),
		fmt.Sprintf("CONTAINER_HOST=%s", c.Host),
		fmt.Sprintf("CONTAINER_ENV=%s", c.Env),
		fmt.Sprintf("HTTP_PORT=%d", c.PrimaryPort),
		fmt.Sprintf("SSHD_PORT=%d", c.SSHPort),
	}
	if c.Manifest.Deps != nil {
		for name, value := range c.Manifest.Deps {
			envs = append(envs, fmt.Sprintf("%s=%s", name, crypto.Decrypt([]byte(value))))
		}
	}

	// get port cfg
	exposedPorts := map[docker.Port]struct{}{}
	portBindings := map[docker.Port][]docker.PortBinding{}
	sPrimaryPort := fmt.Sprintf("%d", c.PrimaryPort)
	dPrimaryPort := docker.NewPort("tcp", sPrimaryPort)
	exposedPorts[dPrimaryPort] = struct{}{}
	portBindings[dPrimaryPort] = []docker.PortBinding{docker.PortBinding{
		HostIp:   "",
		HostPort: fmt.Sprintf("%d", sPrimaryPort),
	}}
	sSSHPort := fmt.Sprintf("%d", c.SSHPort)
	dSSHPort := docker.NewPort("tcp", sSSHPort)
	exposedPorts[dSSHPort] = struct{}{}
	portBindings[dSSHPort] = []docker.PortBinding{docker.PortBinding{
		HostIp:   "",
		HostPort: sSSHPort,
	}}
	for i, port := range c.SecondaryPorts {
		sPort := fmt.Sprintf("%d", port)
		dPort := docker.NewPort("tcp", sPort)
		exposedPorts[dPort] = struct{}{}
		portBindings[dPort] = []docker.PortBinding{docker.PortBinding{
			HostIp:   "",
			HostPort: sPort,
		}}
		envs = append(envs, fmt.Sprintf("SECONDARY_PORT%d=%d", i, port))
	}

	// setup actual cfg
	dCfg := &docker.Config{
		CpuShares:    int64(c.Manifest.CPUShares),
		Memory:       int64(c.Manifest.MemoryLimit) * int64(1024*1024), // this is in bytes
		MemorySwap:   int64(-1),                                        // -1 turns swap off
		ExposedPorts: exposedPorts,
		Env:          envs,
		Cmd:          []string{}, // images already specify run command
		Image:        fmt.Sprintf("%s/%s", RegistryHost, repo),
		Volumes: map[string]struct{}{
			fmt.Sprintf("/var/log/atlantis/containers/%s:/var/log/atlantis/syslog", c.Id): struct{}{},
		},
	}
	dHostCfg := &docker.HostConfig{
		PortBindings: portBindings,
		LxcConf:      []docker.KeyValuePair{},
	}
	return dCfg, dHostCfg
}

// Deploy the given app+sha with the dependencies defined in deps. This will spin up a new docker container.
func (c *Container) Deploy(host, app, sha, env string) error {
	c.Host = host
	c.App = app
	c.Sha = sha
	c.Env = env
	dRepo := fmt.Sprintf("apps/%s-%s", c.App, c.Sha)
	if pretending() {
	} else {
	}
	// Pull docker container
	if pretending() {
		log.Printf("[pretend] deploy %s with %s @ %s...", c.Id, c.App, c.Sha)
		log.Printf("[pretend] docker pull %s/%s", RegistryHost, dRepo)
		log.Printf("[pretend] docker run %s/%s", RegistryHost, dRepo)
		c.DockerId = fmt.Sprintf("pretend-docker-id-%d", c.PrimaryPort)
	} else {
		log.Printf("deploy %s with %s @ %s...", c.Id, c.App, c.Sha)
		dockerLock.Lock()
		err := dockerClient.PullImage(dockercli.PullImageOptions{Repository: dRepo, Registry: RegistryHost},
			os.Stdout)
		dockerLock.Unlock()
		if err != nil {
			return err
		}

		err = os.MkdirAll(fmt.Sprintf("/var/log/atlantis/containers/%s", c.Id), 0755)
		if err != nil {
			return err
		}

		// create docker container
		dCfg, dHostCfg := c.dockerCfgs(dRepo)
		dockerLock.Lock()
		dCont, err := dockerClient.CreateContainer(dockercli.CreateContainerOptions{Name: c.Id}, dCfg)
		dockerLock.Unlock()
		if err != nil {
			return err
		}
		c.DockerId = dCont.ID

		// start docker container
		dockerLock.Lock()
		err = dockerClient.StartContainer(c.DockerId, dHostCfg)
		dockerLock.Unlock()
		if err != nil {
			return err
		}
	}
	save() // save here because this is when we know the deployed container is actually alive
	return nil
}

// Teardown the container. This will kill the docker container but will not free the ports/containers
func (c *Container) teardown() {
	if pretending() {
		log.Printf("[pretend] teardown %s...", c.Id)
		return
	} else {
		log.Printf("teardown %s...", c.Id)
	}
	defer RemoveExited()
	dockerLock.Lock()
	err := dockerClient.KillContainer(c.DockerId)
	dockerLock.Unlock()
	if err != nil {
		log.Printf("failed to teardown %s: %v", c.Id, err)
	}
}

// This calls the Teardown(id string) method to ensure that the ports/containers are freed. That will in turn
// call c.teardown(id string)
func (c *Container) Teardown() {
	Teardown(c.Id)
}

func (c *Container) AuthorizeSSHUser(user, publicKey string) error {
	// copy file to container
	// rebuild authorize_keys
	return SSHCmd{"-p", fmt.Sprintf("%d", c.SSHPort), "-i", "/opt/atlantis/supervisor/master_id_rsa", "-o",
		"UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "root@localhost",
		fmt.Sprintf("echo \"%s\" >/root/.ssh/authorized_keys.d/%s.pub && rebuild_authorized_keys", publicKey,
			user)}.Execute()
}

func (c *Container) DeauthorizeSSHUser(user string) error {
	// delete file from container
	// rebuild authorize_keys
	return SSHCmd{"-p", fmt.Sprintf("%d", c.SSHPort), "-i", "/opt/atlantis/supervisor/master_id_rsa", "-o",
		"UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "root@localhost",
		fmt.Sprintf("rm /root/.ssh/authorized_keys.d/%s.pub && rebuild_authorized_keys",
			user)}.Execute()
}

func (c *Container) SetMaintenance(maint bool) error {
	if maint {
		// touch /etc/maint
		return SSHCmd{"-p", fmt.Sprintf("%d", c.SSHPort), "-i", "/opt/atlantis/supervisor/master_id_rsa", "-o",
			"UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "root@localhost",
			"touch /etc/maint"}.Execute()
	}
	// rm -f /etc/maint
	return SSHCmd{"-p", fmt.Sprintf("%d", c.SSHPort), "-i", "/opt/atlantis/supervisor/master_id_rsa", "-o",
		"UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "root@localhost",
		"rm -f /etc/maint"}.Execute()
}
