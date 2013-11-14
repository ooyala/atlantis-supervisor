package containers

import (
	"atlantis/crypto"
	"atlantis/supervisor/rpc/types"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var dockerIdRegexp = regexp.MustCompile("^[A-Za-z0-9]+$")

type Container types.Container
type DockerCmd []string
type SSHCmd []string

func pretending() bool {
	return os.Getenv("SUPERVISOR_PRETEND") != ""
}

func NewDockerCmd(args ...string) DockerCmd {
	return append([]string{"docker"}, args...)
}

func (d DockerCmd) Execute() (string, error) {
	if pretending() {
		log.Printf("[pretend] docker %s", strings.Join(d, " "))
		return "", nil
	}
	// TODO[jigish] don't print dependencies because they may have passwords!
	log.Printf("sudo %s", strings.Join(d, " "))
	cmd := exec.Command("sudo", d...)
	output, err := cmd.CombinedOutput()
	log.Printf("-> %s", output)
	if err != nil {
		log.Println("-> Error:", err)
	}
	return string(output), err
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

// Deploy the given app+sha with the dependencies defined in deps. This will spin up a new docker container.
func (c *Container) Deploy(host, app, sha, env string) error {
	c.Host = host
	c.App = app
	c.Sha = sha
	c.Env = env
	dockerName := fmt.Sprintf("%s/apps/%s-%s", RegistryHost, c.App, c.Sha)
	if pretending() {
		log.Printf("[pretend] deploy %s with %s @ %s...", c.Id, c.App, c.Sha)
	} else {
		log.Printf("deploy %s with %s @ %s...", c.Id, c.App, c.Sha)
	}
	// Pull docker container
	pullCmd := NewDockerCmd("pull", dockerName)
	_, err := pullCmd.Execute()
	if err != nil {
		return err
	}

	if !pretending() {
		err = os.MkdirAll(fmt.Sprintf("/var/log/atlantis/containers/%s", c.Id), 0755)
		if err != nil {
			return err
		}
	}

	// Run docker container
	runCmd := NewDockerCmd("run", "-d",
		"-name", c.Id,
		"-c", fmt.Sprintf("%d", c.Manifest.CPUShares),
		"-m", fmt.Sprintf("%d", uint64(c.Manifest.MemoryLimit)*uint64(1024*1024)),
		"-v", fmt.Sprintf("/var/log/atlantis/containers/%s:/var/log/atlantis/syslog", c.Id),
		"-e", "ATLANTIS=true",
		"-e", fmt.Sprintf("CONTAINER_ID=%s", c.Id),
		"-e", fmt.Sprintf("CONTAINER_HOST=%s", c.Host),
		"-e", fmt.Sprintf("CONTAINER_ENV=%s", c.Env),
		"-e", fmt.Sprintf("HTTP_PORT=%d", c.PrimaryPort),
		"-e", fmt.Sprintf("SSHD_PORT=%d", c.SSHPort),
		"-p", fmt.Sprintf("%d:%d", c.PrimaryPort, c.PrimaryPort),
		"-p", fmt.Sprintf("%d:%d", c.SSHPort, c.SSHPort))
	for i, port := range c.SecondaryPorts {
		runCmd = append(runCmd, "-e", fmt.Sprintf("SECONDARY_PORT%d=%d", i, port), "-p",
			fmt.Sprintf("%d:%d", port, port))
	}
	if c.Manifest.Deps != nil {
		for name, value := range c.Manifest.Deps {
			runCmd = append(runCmd, "-e", fmt.Sprintf("%s=%s", name, crypto.Decrypt([]byte(value))))
		}
	}
	runCmd = append(runCmd, dockerName)
	dockerId, err := runCmd.Execute()
	if err != nil {
		return err
	}
	// Sometimes docker outputs warnings as well. Kill them here.
	dockerId = strings.TrimSpace(dockerId)
	if idx := strings.LastIndex(dockerId, "\n"); idx >= 0 {
		dockerId = strings.TrimSpace(dockerId[idx:])
	}
	// Set DockerId
	if pretending() {
		c.DockerId = fmt.Sprintf("pretend-docker-id-%d", c.PrimaryPort)
	} else {
		// TODO[jigish] - docker doesn't return proper exit codes yet. make sure the run succeeded here
		if !dockerIdRegexp.MatchString(dockerId) {
			// if docker id is in the wrong format, probably means docker run failed. return an error
			return errors.New("Failed to deploy. Expected [A-Za-z0-9]+ docker id, got '" + dockerId + "'.")
		}
		c.DockerId = dockerId
	}
	save() // save here because this is when we know the deployed container is actually alive
	return nil
}

// Teardown the container. This will kill the docker container but will not free the ports/containers
func (c *Container) teardown() {
	if pretending() {
		log.Printf("[pretend] teardown %s...", c.Id)
	} else {
		log.Printf("teardown %s...", c.Id)
	}
	killCmd := NewDockerCmd("kill", c.Id)
	killCmd.Execute()
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
		fmt.Sprintf("echo \"%s\" >/root/.ssh/authorized_keys.d/%s.pub && oo_rebuild_authorized_keys", publicKey,
			user)}.Execute()
}

func (c *Container) DeauthorizeSSHUser(user string) error {
	// delete file from container
	// rebuild authorize_keys
	return SSHCmd{"-p", fmt.Sprintf("%d", c.SSHPort), "-i", "/opt/atlantis/supervisor/master_id_rsa", "-o",
		"UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "root@localhost",
		fmt.Sprintf("rm /root/.ssh/authorized_keys.d/%s.pub && oo_rebuild_authorized_keys",
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
