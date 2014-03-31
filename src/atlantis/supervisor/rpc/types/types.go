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

package types

import (
	"atlantis/builder/manifest"
	"errors"
	"fmt"
	"strings"
)

type GenericContainer interface {
	GetApp() string
	GetSha() string
	GetID() string
	SetDockerID(string)
	GetDockerID() string
	GetDockerRepo() string
	GetIP() string
	SetIP(string)
	GetPid() int
	SetPid(int)
	GetSSHPort() uint16
}

type Container struct {
	ID             string
	DockerID       string
	IP             string
	Pid            int
	Host           string
	PrimaryPort    uint16
	SecondaryPorts []uint16
	SSHPort        uint16
	App            string
	Sha            string
	Env            string
	Manifest       *Manifest
}

func (c *Container) GetID() string {
	return c.ID
}

func (c *Container) GetApp() string {
	return c.App
}

func (c *Container) GetSha() string {
	return c.Sha
}

func (c *Container) SetDockerID(id string) {
	c.DockerID = id
}

func (c *Container) GetDockerID() string {
	return c.DockerID
}

func (c *Container) GetDockerRepo() string {
	return "apps"
}

func (c *Container) SetIP(ip string) {
	c.IP = ip
}

func (c *Container) GetIP() string {
	return c.IP
}

func (c *Container) SetPid(pid int) {
	c.Pid = pid
}

func (c *Container) GetPid() int {
	return c.Pid
}

func (c *Container) GetSSHPort() uint16 {
	return c.SSHPort
}

func (c *Container) RandomID() string {
	return c.ID[strings.LastIndex(c.ID, "-")+1:]
}

func (c *Container) String() string {
	return fmt.Sprintf(`%s
IP              : %s
Pid             : %d
Host            : %s
Primary Port    : %d
SSH Port        : %d
Secondary Ports : %v
App             : %s
SHA             : %s
CPU Shares      : %d
Memory Limit    : %d
Docker ID       : %s`, c.ID, c.IP, c.Pid, c.Host, c.PrimaryPort, c.SSHPort, c.SecondaryPorts, c.App, c.Sha,
		c.Manifest.CPUShares, c.Manifest.MemoryLimit, c.DockerID)
}

type DepsType map[string]*AppDep
type AppDep struct {
	SecurityGroup map[string][]uint16
	DataMap       map[string]interface{}
	EncryptedData string
}

type Manifest struct {
	Name        string
	Description string
	Instances   uint
	CPUShares   uint
	MemoryLimit uint
	AppType     string
	JavaType    string
	RunCommands []string
	Deps        DepsType
}

func (m *Manifest) Dup() *Manifest {
	runCommands := make([]string, len(m.RunCommands))
	for i, cmd := range m.RunCommands {
		runCommands[i] = cmd
	}
	deps := DepsType{}
	for key, val := range m.Deps {
		deps[key] = &AppDep{
			SecurityGroup: make(map[string][]uint16, len(val.SecurityGroup)),
			DataMap:       map[string]interface{}{},
		}
		for ipGroupName, ports := range val.SecurityGroup {
			if len(ports) == 0 {
				continue
			}
			deps[key].SecurityGroup[ipGroupName] = make([]uint16, len(ports))
			for i, port := range ports {
				deps[key].SecurityGroup[ipGroupName][i] = port
			}
		}
		for innerKey, innerVal := range val.DataMap {
			deps[key].DataMap[innerKey] = innerVal
		}
		deps[key].EncryptedData = val.EncryptedData
	}
	return &Manifest{
		Name:        m.Name,
		Description: m.Description,
		Instances:   m.Instances,
		CPUShares:   m.CPUShares,
		MemoryLimit: m.MemoryLimit,
		AppType:     m.AppType,
		JavaType:    m.JavaType,
		RunCommands: runCommands,
		Deps:        deps,
	}
}

func CreateManifest(mt *manifest.Data) (*Manifest, error) {
	deps := DepsType{}
	for _, name := range mt.Dependencies {
		deps[name] = &AppDep{} // set it here so we can check for it in DepNames()
	}
	var cmds []string
	if len(mt.RunCommands) > 0 {
		cmds = make([]string, len(mt.RunCommands))
		for i, cmd := range mt.RunCommands {
			cmds[i] = cmd
		}
	} else {
		switch runCommand := mt.RunCommand.(type) {
		case string:
			cmds = []string{runCommand}
		case []interface{}:
			cmds = []string{}
			for _, cmd := range runCommand {
				cmdStr, ok := cmd.(string)
				if ok {
					cmds = append(cmds, cmdStr)
				} else {
					return nil, errors.New("Invalid Manifest: non-string element in run_command array!")
				}
			}
		default:
			return nil, errors.New("Invalid Manifest: run_command should be string or []string")
		}
	}
	return &Manifest{
		Name:        mt.Name,
		Description: mt.Description,
		Instances:   0,
		CPUShares:   mt.CPUShares,
		MemoryLimit: mt.MemoryLimit,
		AppType:     mt.AppType,
		JavaType:    mt.JavaType,
		RunCommands: cmds,
		Deps:        deps,
	}, nil
}

func (m *Manifest) DepNames() []string {
	names := make([]string, len(m.Deps))
	i := 0
	for name, _ := range m.Deps {
		names[i] = name
		i++
	}
	return names
}

// ----------------------------------------------------------------------------------------------------------
// Supervisor RPC Types
// ----------------------------------------------------------------------------------------------------------

// ------------ Health Check ------------
// Used to check the health and stats of Supervisor
type SupervisorHealthCheckArg struct {
}

type ResourceStats struct {
	Total uint
	Used  uint
	Free  uint
}

type SupervisorHealthCheckReply struct {
	Containers *ResourceStats
	CPUShares  *ResourceStats
	Memory     *ResourceStats
	Region     string
	Zone       string
	Status     string
}

// ------------ Deploy ------------
// Used to deploy a new app/sha
type SupervisorDeployArg struct {
	Host        string
	App         string
	Sha         string
	Env         string
	ContainerID string
	Manifest    *Manifest
}

type SupervisorDeployReply struct {
	Status    string
	Container *Container
}

// ------------ Teardown ------------
// Used to teardown a container
type SupervisorTeardownArg struct {
	ContainerIDs []string
	All          bool
}

type SupervisorTeardownReply struct {
	ContainerIDs []string
	Status       string
}

// ------------ Get ------------
// Used to get a container
type SupervisorGetArg struct {
	ContainerID string
}

type SupervisorGetReply struct {
	Container *Container
	Status    string
}

// ------------ List ------------
// List Supervisor Containers
type SupervisorListArg struct {
}

type SupervisorListReply struct {
	Containers  map[string]*Container
	UnusedPorts []uint16
}

// ------------ Authorize SSH ------------
// Authorize SSH
type SupervisorAuthorizeSSHArg struct {
	ContainerID string
	User        string
	PublicKey   string
}

type SupervisorAuthorizeSSHReply struct {
	Port   uint16
	Status string
}

// ------------ Deauthorize SSH ------------
// Deauthorize SSH
type SupervisorDeauthorizeSSHArg struct {
	ContainerID string
	User        string
}

type SupervisorDeauthorizeSSHReply struct {
	Status string
}

// ------------ Update IP Group ------------
type SupervisorUpdateIPGroupArg struct {
	Name string
	IPs  []string
}

type SupervisorUpdateIPGroupReply struct {
	Status string
}

// ------------ Delete IP Group ------------
type SupervisorDeleteIPGroupArg struct {
	Name string
}

type SupervisorDeleteIPGroupReply struct {
	Status string
}

// ------------ Container Maintenance ------------
// Set Container Maintenance Mode
type SupervisorContainerMaintenanceArg struct {
	ContainerID string
	Maintenance bool
}

type SupervisorContainerMaintenanceReply struct {
	Status string
}

// ------------ Idle ------------
// Check if Idle
type SupervisorIdleArg struct {
}

type SupervisorIdleReply struct {
	Idle   bool
	Status string
}
