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

package monitor

import (
	"atlantis/supervisor/containers/serialize"
	"atlantis/supervisor/rpc/types"
	"bytes"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/jigish/go-flags"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	OK = iota
	Warning
	Critical
	Uknown
)

type Config struct {
	ContainerFile   string `toml:"container_file"`
	ContainersDir   string `toml:"container_dir"`
	InventoryDir    string `toml:"inventory_dir"`
	SSHIdentity     string `toml:"ssh_identity"`
	SSHUser         string `toml:"ssh_user"`
	CheckName       string `toml:"check_name"`
	CheckDir        string `toml:"check_dir"`
	DefaultGroup    string `toml:"default_group"`
	TimeoutDuration uint   `toml:"timeout_duration"`
	Verbose         bool   `toml:"verbose"`
}

type Opts struct {
	ContainerFile   string `short:"f" long:"container-file" description:"file to get container information"`
	ContainersDir   string `short:"s" long:"containers-dir" description:"directory containing configs for each container"`
	SSHIdentity     string `short:"i" long:"ssh-identity" description:"file containing the SSH key for all containers"`
	SSHUser         string `short:"u" long:"ssh-user" description:"user account to ssh into containers"`
	CheckName       string `short:"n" long:"check-name" description:"service name that will appear in Nagios for the monitor"`
	CheckDir        string `short:"d" long:"check-dir" description:"directory containing all the scripts for the monitoring checks"`
	DefaultGroup    string `short:"g" long:"default-group" description:"default contact group to use if there is no valid group provided"`
	Config          string `short:"c" long:"config-file" default:"/etc/atlantis/supervisor/monitor.toml" description:"the config file to use"`
	TimeoutDuration uint   `short:"t" long:"timeout-duration" description:"max number of seconds to wait for a monitoring check to finish"`
	Verbose         bool   `short:"v" long:"verbose" default:false description:"print verbose debug information"`
}

type ServiceCheck struct {
	Service  string
	User     string
	Identity string
	Host     string
	Port     uint16
	Script   string
}

//TODO(mchandra):Need defaults defined by constants
var config = &Config{
	ContainerFile:   "/etc/atlantis/supervisor/save/containers",
	ContainersDir:   "/etc/atlantis/containers",
	InventoryDir:    "/etc/atlantis/supervisor/inventory",
	SSHIdentity:     "/opt/atlantis/supervisor/master_id_rsa",
	SSHUser:         "root",
	CheckName:       "ContainerMonitor",
	CheckDir:        "/check_mk_checks",
	DefaultGroup:    "atlantis_orphan_apps",
	TimeoutDuration: 110,
	Verbose:         false,
}

func (s *ServiceCheck) cmd() *exec.Cmd {
	return silentSshCmd(s.User, s.Identity, s.Host, s.Script, s.Port)
}

func (s *ServiceCheck) timeOutMsg() string {
	return fmt.Sprintf("%d %s - Timeout occured during check\n", Critical, s.Service)
}

func (s *ServiceCheck) errMsg(err error) string {
	if err != nil {
		return fmt.Sprintf("%d %s - %s\n", Critical, s.Service, err.Error())
	} else {
		return fmt.Sprintf("%d %s - Error encountered while monitoring the service\n", Critical, s.Service)
	}
}

func (s *ServiceCheck) validate(msg string) string {
	m := strings.SplitN(msg, " ", 4)
	if len(m) > 1 && m[1] == s.Service {
		return msg
	}
	return s.errMsg(nil)
}

func (s *ServiceCheck) runCheck(done chan bool) {
	out, err := s.cmd().Output()
	if err != nil {
		fmt.Print(s.errMsg(err))
	} else {
		fmt.Print(s.validate(string(out)))
	}
	done <- true
}

func (s *ServiceCheck) checkWithTimeout(results chan bool, d time.Duration) {
	done := make(chan bool, 1)
	go s.runCheck(done)
	select {
	case <-done:
		results <- true
	case <-time.After(d):
		fmt.Print(s.timeOutMsg())
		results <- true
	}
}

type ContainerCheck struct {
	Name         string
	User         string
	Identity     string
	Directory    string
	Inventory    string
	ContactGroup string
	container    *types.Container
}

type ContainerConfig struct {
	Dependencies map[string]interface{}
}

func (c *ContainerCheck) verifyContactGroup(group string) bool {
	output, err := exec.Command("/usr/bin/cmk_admin", "-l").Output()
	for _, l := range strings.Split(output, "\n") {
		cg := strings.TrimSpace(strings.TrimPrefix(l, "*"))
		if cg == c.ContactGroup {
			return true
		}
	}
	return false
}

func (c *ContainerCheck) getContactGroup() {
	c.ContactGroup = config.DefaultGroup
	config_file := filepath.Join(config.ContainersDir, c.container.ID, "config.json")
	var cont_config ContainerConfig
	if err := serialize.RetrieveObject(config_file, &cont_config); err != nil {
		fmt.Printf("%d %s - Could not retrieve container config %s: %s\n", Critical, config.CheckName, config_file, err)
	} else {
		dep, ok := cont_config.Dependencies["cmk"]
		if !ok {
			fmt.Printf("%d %s - cmk dep not present, defaulting to %s contact group!\n", OK, config.CheckName, config.DefaultGroup)
			return
		}
		cmk_dep, ok := dep.(map[string]interface{})
		if !ok {
			fmt.Printf("%d %s - cmk dep present, but value is not map[string]string!\n", Critical, config.CheckName)
			return
		}
		val, ok := cmk_dep["contact_group"]
		if !ok {
			fmt.Printf("%d %s - cmk dep present, but no contact_group key!\n", Critical, config.CheckName)
			return
		}
		group, ok := val.(string)
		if ok {
			group = strings.ToLower(group)
			if c.verifyContactGroup(group) {
				c.ContactGroup = group
			} else {
				fmt.Printf("%d %s - Specified contact_group does not exist in cmk! Falling back to default group %s.\n", Critical, config.CheckName, config.DefaultGroup)
			}
		} else {
			fmt.Printf("%d %s - Value for contact_group key of cmk dep is not a string!\n", Critical, config.CheckName)
		}
	}
}

func (c *ContainerCheck) setContactGroup(name string) {
	if len(c.ContactGroup) == 0 {
		c.getContactGroup()
	}
	inventoryPath := path.Join(c.Inventory, name)
	if _, err := os.Stat(inventoryPath); os.IsNotExist(err) {
		output, err := exec.Command("/usr/bin/cmk_admin", "-s", name, "-a", c.ContactGroup).CombinedOutput()
		if err != nil {
			fmt.Printf("%d %s - Failure to update contact group for service %s. Error: %s\n", OK, config.CheckName, name, err.Error())
		} else {
			os.Create(inventoryPath)
		}
		if config.Verbose {
			fmt.Printf("\n/usr/bin/cmk_admin -s %s -a %s\n%s\n\n", name, c.ContactGroup, output)
		}
	}
}

func (c *ContainerCheck) Run(t time.Duration, done chan bool) {
	c.setContactGroup(c.Name)
	defer func() { done <- true }()
	o, err := silentSshCmd(c.User, c.Identity, c.container.Host, "ls "+c.Directory, c.container.SSHPort).Output()
	if err != nil {
		fmt.Printf("%d %s - Error getting checks for container: %s\n", Critical, c.Name, err.Error())
		return
	}
	fmt.Printf("%d %s - Got checks for container\n", OK, c.Name)
	scripts := strings.Split(strings.TrimSpace(string(o)), "\n")
	if len(scripts) == 0 || len(scripts[0]) == 0 {
		// nothing to check on this container, exit
		return
	}
	c.checkAll(scripts, t)
}

func (c *ContainerCheck) checkAll(scripts []string, t time.Duration) {
	results := make(chan bool, len(scripts))
	for _, s := range scripts {
		serviceName := fmt.Sprintf("%s_%s", strings.Split(s, ".")[0], c.container.ID)
		c.setContactGroup(serviceName)
		go c.serviceCheck(s).checkWithTimeout(results, t)
	}
	for _ = range scripts {
		<-results
	}
}

func (c *ContainerCheck) serviceCheck(script string) *ServiceCheck {
	// The full path to the script is required
	command := fmt.Sprintf("%s/%s %d %s", c.Directory, script, c.container.PrimaryPort, c.container.ID)
	// The service name is obtained be removing the file extension from the script and appending the container
	// id
	serviceName := fmt.Sprintf("%s_%s", strings.Split(script, ".")[0], c.container.ID)
	return &ServiceCheck{serviceName, c.User, c.Identity, c.container.Host, c.container.SSHPort, command}
}

func silentSshCmd(user, identity, host, cmd string, port uint16) *exec.Cmd {
	args := []string{"-q", user + "@" + host, "-i", identity, "-p", fmt.Sprintf("%d", port), "-o", "StrictHostKeyChecking=no", cmd}
	return exec.Command("ssh", args...)
}

func overlayConfig() {
	opts := &Opts{}
	flags.Parse(opts)
	if opts.Config != "" {
		_, err := toml.DecodeFile(opts.Config, config)
		if err != nil {
			// no need to panic here. we have reasonable defaults.
		}
	}
	if opts.ContainerFile != "" {
		config.ContainerFile = opts.ContainerFile
	}
	if opts.ContainersDir != "" {
		config.ContainersDir = opts.ContainersDir
	}
	if opts.SSHIdentity != "" {
		config.SSHIdentity = opts.SSHIdentity
	}
	if opts.SSHUser != "" {
		config.SSHUser = opts.SSHUser
	}
	if opts.CheckDir != "" {
		config.CheckDir = opts.CheckDir
	}
	if opts.CheckName != "" {
		config.CheckName = opts.CheckName
	}
	if opts.DefaultGroup != "" {
		config.DefaultGroup = opts.DefaultGroup
	}
	if opts.TimeoutDuration != 0 {
		config.TimeoutDuration = opts.TimeoutDuration
	}
	if opts.Verbose {
		config.Verbose = true
	}
}

//file containing containers and service name to show in Nagios for the monitor itself
func Run() {
	overlayConfig()
	var contMap map[string]*types.Container
	//Check if folder exists
	_, err := os.Stat(config.ContainerFile)
	if os.IsNotExist(err) {
		fmt.Printf("%d %s - Container file does not exists %s. Likely no live containers present.\n", OK, config.CheckName, config.ContainerFile)
		return
	}
	if err := serialize.RetrieveObject(config.ContainerFile, &contMap); err != nil {
		fmt.Printf("%d %s - Error retrieving %s: %s\n", Critical, config.CheckName, config.ContainerFile, err)
		return
	}
	done := make(chan bool, len(contMap))
	config.SSHIdentity = strings.Replace(config.SSHIdentity, "~", os.Getenv("HOME"), 1)
	for _, c := range contMap {
		if c.Host == "" {
			c.Host = "localhost"
		}
		check := &ContainerCheck{config.CheckName + "_" + c.ID, config.SSHUser, config.SSHIdentity, config.CheckDir, config.InventoryDir, "", c}
		go check.Run(time.Duration(config.TimeoutDuration)*time.Second, done)
	}
	for _ = range contMap {
		<-done
	}
	// Clean up inventories from containers that no longer exist
	err = filepath.Walk(config.InventoryDir, func(path string, _ os.FileInfo, _ error) error {
		if path == config.InventoryDir {
			return nil
		}
		var err error
		split := strings.Split(path, "_")
		cont := split[len(split)-1]
		if _, ok := contMap[cont]; !ok {
			err = os.Remove(path)
		}
		return err
	})
	if err != nil {
		fmt.Printf("%d %s - Error iterating over inventory to delete obsolete markers. Error: %s\n", OK, config.CheckName, err.Error())
	}
}
