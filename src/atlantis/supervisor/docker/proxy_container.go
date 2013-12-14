package docker

import (
	"atlantis/supervisor/rpc/types"
	"fmt"
	"github.com/jigish/go-dockerclient"
)

func ProxyContainerDockerCfgs(c *types.ProxyContainer) (*docker.Config, *docker.HostConfig) {
	// get env cfg
	envs := []string{
		"ATLANTIS=true",
		fmt.Sprintf("CONTAINER_ID=%s", c.ID),
		fmt.Sprintf("CONTAINER_HOST=%s", c.Host),
		fmt.Sprintf("CONFIG_PORT=%d", c.ConfigPort),
		fmt.Sprintf("SSHD_PORT=%d", c.SSHPort),
		fmt.Sprintf("PROXY_NUM_HANDLERS=%d", c.NumHandlers),
		fmt.Sprintf("PROXY_MAX_PENDING=%d", c.MaxPending),
	}

	// get port cfg
	exposedPorts := map[docker.Port]struct{}{}
	portBindings := map[docker.Port][]docker.PortBinding{}
	sConfigPort := fmt.Sprintf("%d", c.ConfigPort)
	dConfigPort := docker.NewPort("tcp", sConfigPort)
	exposedPorts[dConfigPort] = struct{}{}
	portBindings[dConfigPort] = []docker.PortBinding{docker.PortBinding{
		HostIp:   "",
		HostPort: sConfigPort,
	}}
	sSSHPort := fmt.Sprintf("%d", c.SSHPort)
	dSSHPort := docker.NewPort("tcp", sSSHPort)
	exposedPorts[dSSHPort] = struct{}{}
	portBindings[dSSHPort] = []docker.PortBinding{docker.PortBinding{
		HostIp:   "",
		HostPort: sSSHPort,
	}}
	for port := uint16(c.MinExposePort); port <= c.MaxExposePort; port++ {
		sPort := fmt.Sprintf("%d", port)
		dPort := docker.NewPort("tcp", sPort)
		exposedPorts[dPort] = struct{}{}
	}

	// setup actual cfg
	dCfg := &docker.Config{
		CpuShares:    c.CPUShares,
		Memory:       c.MemoryLimit * int64(1024*1024), // this is in bytes
		MemorySwap:   int64(-1),                        // -1 turns swap off
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
				Value: c.VEthName,
			},
		},
	}
	return dCfg, dHostCfg
}
