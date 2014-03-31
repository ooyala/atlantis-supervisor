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

package containers

import (
	"atlantis/supervisor/docker"
	"atlantis/supervisor/rpc/types"
)

type Container struct {
	types.Container
}

// Deploy the given app+sha with the dependencies defined in deps. This will spin up a new docker container.
func (c *Container) Deploy(host, app, sha, env string) error {
	c.Host = host
	c.App = app
	c.Sha = sha
	c.Env = env
	err := docker.Deploy(&c.Container)
	if err != nil {
		return err
	}
	// by this time Pid should be filled in
	NetworkSecurity.AddContainerSecurity(c.ID, c.Pid, c.getSecurityGroups()) // add network security
	save()                                                                   // save here because this is when we know the deployed container is actually alive
	inventory()                                                              // now that the container is up and we've saved it, inventory check_mk
	return nil
}

func (c *Container) getSecurityGroups() map[string][]uint16 {
	sgsMap := map[string]map[uint16]bool{}
	for _, appDep := range c.Manifest.Deps {
		for name, ports := range appDep.SecurityGroup {
			if len(ports) == 0 {
				continue
			}
			// fill in to a map first to dedup
			if len(sgsMap[name]) == 0 {
				sgsMap[name] = map[uint16]bool{}
			}
			for _, port := range ports {
				sgsMap[name][port] = true
			}
		}
	}
	sgs := map[string][]uint16{}
	for name, portMap := range sgsMap {
		if len(portMap) == 0 {
			continue
		}
		sgs[name] = make([]uint16, len(portMap))
		i := 0
		for port, _ := range portMap {
			sgs[name][i] = port
			i++
		}
	}
	return sgs
}

// This calls the Teardown(id string) method to ensure that the ports/containers are freed. That will in turn
// call c.teardown(id string)
func (c *Container) Teardown() {
	NetworkSecurity.RemoveContainerSecurity(c.ID)
	Teardown(c.ID)
}
