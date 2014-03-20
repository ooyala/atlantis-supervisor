/* Copyright 2014 Ooyala, Inc. All rights reserved.
 *
 * This file is licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package docker

import (
	. "atlantis/supervisor/constant"
	"atlantis/supervisor/crypto"
	"atlantis/supervisor/helper"
	"atlantis/supervisor/rpc/types"
	atypes "atlantis/types"
	"fmt"
	"github.com/fsouza/go-dockerclient"
)

func NewDockerPort(port, proto string) docker.Port {
	return docker.Port(fmt.Sprintf("%s/%s", port, proto))
}

func ContainerAppCfgs(c *types.Container) (*atypes.AppConfig, error) {
	var err error
	deps := map[string]map[string]interface{}{}
	if c.Manifest.Deps != nil {
		for name, appDep := range c.Manifest.Deps {
			deps[name], err = crypto.DecryptedAppDepData(appDep)
			if err != nil {
				return nil, err
			}
		}
	}
	return &atypes.AppConfig{
		HTTPPort:       c.PrimaryPort,
		SecondaryPorts: c.SecondaryPorts,
		Container: &atypes.ContainerConfig{
			ID:   c.ID,
			Host: c.Host,
			Env:  c.Env,
		},
		Dependencies: deps,
	}, nil
}

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

	// get port cfg
	exposedPorts := map[docker.Port]struct{}{}
	portBindings := map[docker.Port][]docker.PortBinding{}
	sPrimaryPort := fmt.Sprintf("%d", c.PrimaryPort)
	dPrimaryPort := NewDockerPort(sPrimaryPort, "tcp")
	exposedPorts[dPrimaryPort] = struct{}{}
	portBindings[dPrimaryPort] = []docker.PortBinding{docker.PortBinding{
		HostIp:   "",
		HostPort: sPrimaryPort,
	}}
	sSSHPort := fmt.Sprintf("%d", c.SSHPort)
	dSSHPort := NewDockerPort(sSSHPort, "tcp")
	exposedPorts[dSSHPort] = struct{}{}
	portBindings[dSSHPort] = []docker.PortBinding{docker.PortBinding{
		HostIp:   "",
		HostPort: sSSHPort,
	}}
	for i, port := range c.SecondaryPorts {
		sPort := fmt.Sprintf("%d", port)
		dPort := NewDockerPort(sPort, "tcp")
		exposedPorts[dPort] = struct{}{}
		portBindings[dPort] = []docker.PortBinding{docker.PortBinding{
			HostIp:   "",
			HostPort: sPort,
		}}
		envs = append(envs, fmt.Sprintf("SECONDARY_PORT%d=%d", i, port))
	}

	// setup actual cfg
	dCfg := &docker.Config{
		Tty:          true, // allocate pseudo-tty
		OpenStdin:    true, // keep stdin open even if we're not attached
		CpuShares:    int64(c.Manifest.CPUShares),
		Memory:       int64(c.Manifest.MemoryLimit) * int64(1024*1024), // this is in bytes
		MemorySwap:   int64(-1),                                        // -1 turns swap off
		ExposedPorts: exposedPorts,
		Env:          envs,
		Cmd: []string{
			"runsvdir",
			"/etc/service",
		},
		Image: fmt.Sprintf("%s/%s/%s-%s", RegistryHost, c.GetDockerRepo(), c.App, c.Sha),
		Volumes: map[string]struct{}{
			ContainerLogDir:           struct{}{},
			atypes.ContainerConfigDir: struct{}{},
		},
	}
	dHostCfg := &docker.HostConfig{
		PortBindings: portBindings,
		Binds: []string{
			fmt.Sprintf("%s:%s", helper.HostLogDir(c.ID), ContainerLogDir),
			fmt.Sprintf("%s:%s", helper.HostConfigDir(c.ID), atypes.ContainerConfigDir),
		},
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
