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
	// TODO update iptables
	// sudo iptables -A PREROUTING -t mangle -m physdev --physdev-in veth<container> -j MARK --set-mark <container mark>
	// sudo iptables -I FORWARD -p tcp <match destination ip of router and destination ports that we want> -j ACCEPT
	// iptables to kill all traffic out of the container
	save() // save here because this is when we know the deployed container is actually alive
	return nil
}

// This calls the Teardown(id string) method to ensure that the ports/containers are freed. That will in turn
// call c.teardown(id string)
func (c *Container) Teardown() {
	// TODO update iptables
	Teardown(c.ID)
}
