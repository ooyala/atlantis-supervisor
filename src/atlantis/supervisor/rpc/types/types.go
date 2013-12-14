package types

import (
	ptypes "atlantis/proxy/types"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
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
	GetSSHPort() uint16
}

type ProxyContainer struct {
	ID            string
	DockerID      string
	IP            string
	Host          string
	ConfigPort    uint16
	SSHPort       uint16
	MinExposePort uint16
	MaxExposePort uint16
	App           string
	Sha           string
	NumHandlers   int
	MaxPending    int
	CPUShares     int64
	MemoryLimit   int64
	VEthName      string
}

func (c *ProxyContainer) GetID() string {
	return c.ID
}

func (c *ProxyContainer) GetApp() string {
	return c.App
}

func (c *ProxyContainer) GetSha() string {
	return c.Sha
}

func (c *ProxyContainer) SetDockerID(id string) {
	c.DockerID = id
}

func (c *ProxyContainer) GetDockerID() string {
	return c.DockerID
}

func (c *ProxyContainer) GetDockerRepo() string {
	return "system"
}

func (c *ProxyContainer) SetIP(ip string) {
	c.IP = ip
}

func (c *ProxyContainer) GetIP() string {
	return c.IP
}

func (c *ProxyContainer) GetSSHPort() uint16 {
	return c.SSHPort
}

func (c *ProxyContainer) String() string {
	return fmt.Sprintf(`%s
IP              : %s
Host            : %s
Config Port     : %d
SSH Port        : %d
App             : %s
SHA             : %s
Min Expose Port : %d
Max Expose Port : %d
Docker ID       : %s`, c.ID, c.IP, c.Host, c.ConfigPort, c.SSHPort, c.App, c.Sha, c.MinExposePort,
		c.MaxExposePort, c.DockerID)
}

type Container struct {
	ID             string
	DockerID       string
	IP             string
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

func (c *Container) GetSSHPort() uint16 {
	return c.SSHPort
}

func (c *Container) RandomID() string {
	return c.ID[strings.LastIndex(c.ID, "-")+1:]
}

func (c *Container) String() string {
	return fmt.Sprintf(`%s
IP              : %s
Host            : %s
Primary Port    : %d
SSH Port        : %d
Secondary Ports : %v
App             : %s
SHA             : %s
CPU Shares      : %d
Memory Limit    : %d
Docker ID       : %s`, c.ID, c.IP, c.Host, c.PrimaryPort, c.SSHPort, c.SecondaryPorts, c.App, c.Sha,
		c.Manifest.CPUShares, c.Manifest.MemoryLimit, c.DockerID)
}

// NOTE[jigish]: ONLY for TOML parsing
type ManifestTOML struct {
	Name        string
	Description string
	Internal    bool
	Instances   uint
	CPUShares   uint `toml:"cpu_shares"`   // should be 1 or any multiple of 5
	MemoryLimit uint `toml:"memory_limit"` // should be a multiple of 256 (MBytes)
	Image       string
	AppType     string      `toml:"app_type"`
	RunCommand  interface{} `toml:"run_command"` // can be string or array
	DepNames    []string    `toml:"dependencies"`
}

type Manifest struct {
	Name        string
	Description string
	Internal    bool
	Instances   uint
	CPUShares   uint
	MemoryLimit uint
	Image       string
	AppType     string
	RunCommands []string
	Deps        map[string]string
}

func (m *Manifest) Dup() *Manifest {
	runCommands := make([]string, len(m.RunCommands))
	for i, cmd := range m.RunCommands {
		runCommands[i] = cmd
	}
	deps := map[string]string{}
	for key, val := range m.Deps {
		deps[key] = val
	}
	return &Manifest{
		Name:        m.Name,
		Description: m.Description,
		Internal:    m.Internal,
		Instances:   m.Instances,
		CPUShares:   m.CPUShares,
		MemoryLimit: m.MemoryLimit,
		Image:       m.Image,
		AppType:     m.AppType,
		RunCommands: runCommands,
		Deps:        deps,
	}
}

func CreateManifest(mt *ManifestTOML) (*Manifest, error) {
	deps := map[string]string{}
	for _, name := range mt.DepNames {
		deps[name] = "" // set it here so we can check for it in DepNames()
	}
	var cmds []string
	switch runCommand := mt.RunCommand.(type) {
	case string:
		cmds = []string{runCommand}
	case []interface{}:
		cmds = make([]string, 1, 1)
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
	return &Manifest{
		Name:        mt.Name,
		Description: mt.Description,
		Internal:    mt.Internal,
		Instances:   mt.Instances,
		CPUShares:   mt.CPUShares,
		MemoryLimit: mt.MemoryLimit,
		Image:       mt.Image,
		AppType:     mt.AppType,
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

func ReadManifest(r io.Reader) (*Manifest, error) {
	var manifestTOML ManifestTOML
	_, err := toml.DecodeReader(r, &manifestTOML)
	if err != nil {
		return nil, errors.New("Parse Manifest Error: " + err.Error())
	}
	return CreateManifest(&manifestTOML)
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

// ------------ UpdateProxy ------------
// UpdateProxy
type SupervisorUpdateProxyArg struct {
	Host string
	Sha  string
}

type SupervisorUpdateProxyReply struct {
	Status string
	Proxy  *ProxyContainer
}

// ------------ GetProxy ------------
// GetProxy
type SupervisorGetProxyArg struct {
}

type SupervisorGetProxyReply struct {
	Proxy *ProxyContainer
}

// ------------ ConfigureProxy ------------
// ConfigureProxy
type SupervisorConfigureProxyArg struct {
	ProxyConfig map[string]*ptypes.ProxyConfig
}

type SupervisorConfigureProxyReply struct {
	Status string
}
