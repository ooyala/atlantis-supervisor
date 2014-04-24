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

package netsec

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func guano(pid int) (string, string, error) {
	binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", "", err
	}
	out, err := executeCommand(path.Join(binDir, "guano"), fmt.Sprintf("%d", pid), "eth0")
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(out, " ")
	if len(parts) != 2 {
		return "", "", errors.New("guano is batshit insane.")
	}
	return parts[0], parts[1], nil
}

type ContainerSecurity struct {
	veth           string
	mark           string
	ID             string
	Pid            int
	SecurityGroups map[string][]uint16 // ipgroup name -> ports
}

func NewContainerSecurity(id string, pid int, sgs map[string][]uint16) (contSec *ContainerSecurity, err error) {
	contSec = &ContainerSecurity{
		ID:             id,
		Pid:            pid,
		SecurityGroups: sgs,
	}
	contSec.mark, contSec.veth, err = guano(pid)
	return contSec, err
}

func (c *ContainerSecurity) filterPort(action, ip string, port uint16) error {
	_, err := executeCommand("iptables", action, "FORWARD",
		"-d", ip,
		"-p", "tcp", "--dport", fmt.Sprintf("%d", port),
		"-m", "mark", "--mark", c.mark,
		"-j", "ACCEPT")
	return err
}

func (c *ContainerSecurity) allowPort(ip string, port uint16) error {
	return c.filterPort("-I", ip, port)
}

func (c *ContainerSecurity) rejectPort(ip string, port uint16) error {
	return c.filterPort("-D", ip, port)
}

func (c *ContainerSecurity) markVeth(action string) error {
	_, err := executeCommand("iptables", action, "PREROUTING", "-t", "mangle",
		"-m", "physdev", "--physdev-in", c.veth,
		"-j", "MARK", "--set-mark", c.mark)
	return err
}

func (c *ContainerSecurity) addMark() error {
	return c.markVeth("-I")
}

func (c *ContainerSecurity) delMark() error {
	return c.markVeth("-D")
}
