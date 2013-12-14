package docker

import (
	"atlantis/crypto"
	"atlantis/supervisor/rpc/types"
	"fmt"
	"github.com/jigish/go-dockerclient"
)

func ContainerDockerCfgs(c *types.Container) (*docker.Config, *docker.HostConfig) {
	// get env cfg
	envs := []string{
		"ATLANTIS=true",
		fmt.Sprintf("CONTAINER_ID=%s", c.ID),
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
		HostPort: sPrimaryPort,
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
		Image:        fmt.Sprintf("%s/%s/%s-%s", RegistryHost, c.GetDockerRepo(), c.App, c.Sha),
		Volumes:      map[string]struct{}{"/var/log/atlantis/syslog": struct{}{}},
	}
	dHostCfg := &docker.HostConfig{
		PortBindings: portBindings,
		Binds:        []string{fmt.Sprintf("/var/log/atlantis/containers/%s:/var/log/atlantis/syslog", c.ID)},
		// call veth something we can actually look up later:
		LxcConf: []docker.KeyValuePair{
			docker.KeyValuePair{
				Key:   "lxc.network.veth.pair",
				Value: "veth" + c.RandomID(),
			},
		},
	}
	return dCfg, dHostCfg
}
