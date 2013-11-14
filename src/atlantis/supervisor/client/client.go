package client

import (
	. "atlantis/common"
	. "atlantis/supervisor/constant"
	. "atlantis/supervisor/rpc/types"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/jigish/go-flags"
	"log"
	"time"
)

type Config struct {
	Host string `toml:"host"`
	Port uint16 `toml:"port"`
}

func (c *Config) RPCHostAndPort() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type Opts struct {
	Host   string `short:"H" long:"host" description:"the supervisor host to use"`
	Port   uint16 `short:"P" long:"port" description:"the supervisor port to use"`
	Config string `short:"F" long:"config-file" default:"/etc/atlantis/supervisor/client.toml" description:"the config file to use"`
}

var opts = &Opts{}
var config = &Config{"localhost", DefaultSupervisorRPCPort}
var rpcClient = NewRPCClientWithConfig(config, "Supervisor", SupervisorRPCVersion, false)

type Supervisor struct {
	*flags.Parser
}

// Creates and returns a new Supervisor
func New() *Supervisor {
	ih := &Supervisor{flags.NewParser(opts, flags.Default)}
	ih.AddCommand("health", "check supervisor's health", "", &HealthCommand{})
	ih.AddCommand("list", "list supervisor containers & unused ports", "", &ListCommand{})
	ih.AddCommand("deploy", "deploy an app+sha", "", &DeployCommand{})
	ih.AddCommand("teardown", "teardown one or more containers", "", &TeardownCommand{})
	ih.AddCommand("get", "get information about a container", "", &GetCommand{})
	ih.AddCommand("version", "check supervisor's client and server versions", "", &VersionCommand{})
	ih.AddCommand("authorize-ssh", "authorize ssh into a container", "", &AuthorizeSSHCommand{})
	ih.AddCommand("deuthorize-ssh", "deauthorize ssh access to a container", "", &DeauthorizeSSHCommand{})
	ih.AddCommand("container-maintenance", "set maintenance mode for a container", "",
		&ContainerMaintenanceCommand{})
	ih.AddCommand("idle", "check if supervisor is idle", "", &IdleCommand{})
	return ih
}

// Runs the  Passing -d as the first flag will run the server, otherwise the client is run.
func (ih *Supervisor) Run() {
	ih.Parse()
}

func overlayConfig() {
	if opts.Config != "" {
		_, err := toml.DecodeFile(opts.Config, config)
		if err != nil {
			log.Println(err)
			// no need to panic here. we have reasonable defaults.
		}
	}
	if opts.Host != "" {
		config.Host = opts.Host
	}
	if opts.Port != 0 {
		config.Port = opts.Port
	}
}

// ----------------------------------------------------------------------------------------------------------
// Client Methods
// ----------------------------------------------------------------------------------------------------------

type HealthCommand struct {
}

func (c *HealthCommand) Execute(args []string) error {
	overlayConfig()
	log.Println("Supervisor Health Check...")
	arg := SupervisorHealthCheckArg{}
	var reply SupervisorHealthCheckReply
	err := rpcClient.Call("HealthCheck", arg, &reply)
	if err != nil {
		return err
	}
	log.Printf("-> region: %s", reply.Region)
	log.Printf("-> zone: %s", reply.Zone)
	log.Printf("-> containers: %d total, %d used, %d free", reply.Containers.Total, reply.Containers.Used,
		reply.Containers.Free)
	log.Printf("-> cpu shares: %d total, %d used, %d free", reply.CPUShares.Total, reply.CPUShares.Used,
		reply.CPUShares.Free)
	log.Printf("-> memory: %d MB total, %d MB used, %d MB free", reply.Memory.Total, reply.Memory.Used,
		reply.Memory.Free)
	log.Printf("-> status: %s", reply.Status)
	return nil
}

type ListCommand struct {
}

func (c *ListCommand) Execute(args []string) error {
	overlayConfig()
	log.Println("Supervisor List...")
	arg := SupervisorListArg{}
	var reply SupervisorListReply
	err := rpcClient.Call("List", arg, &reply)
	if err != nil {
		return err
	}
	log.Printf("-> UnusedPorts: %v", reply.UnusedPorts)
	log.Println("-> Containers:")
	for _, cont := range reply.Containers {
		log.Println("-> " + cont.String())
	}
	return nil
}

type DeployCommand struct {
	Host        string            `short:"H" long:"host" description:"the host we're deploying on"`
	App         string            `short:"a" long:"app" description:"the app to deploy"`
	Sha         string            `short:"s" long:"sha" description:"the sha to deploy"`
	Env         string            `short:"e" long:"env" description:"the env to deploy"`
	Container   string            `short:"c" long:"container" description:"the container id to deploy"`
	CPUShares   uint              `short:"C" long:"cpu-shares" description:"the number of cpu shares to use"`
	MemoryLimit uint              `short:"m" long:"memory-limit" description:"the MBytes of memory to use"`
	Deps        map[string]string `short:"d" long:"dep" description:"specify a service dependency"`
}

func (c *DeployCommand) Execute(args []string) error {
	overlayConfig()
	if c.App == "" {
		return errors.New("Please specify an app")
	}
	if c.Sha == "" {
		return errors.New("Please specify a sha")
	}
	if c.Container == "" {
		c.Container = fmt.Sprintf("%s-%s-%s-%d", c.App, c.Sha, config.Host, time.Now().Unix())
	}
	log.Printf("Supervisor Deploy %s @ %s -> %s...", c.App, c.Sha, c.Container)
	manifest := &Manifest{}
	manifest.Deps = c.Deps
	manifest.CPUShares = c.CPUShares
	manifest.MemoryLimit = c.MemoryLimit
	log.Printf("-> Dependencies: %#v", manifest.Deps)
	arg := SupervisorDeployArg{c.Host, c.App, c.Sha, c.Env, c.Container, manifest}
	var reply SupervisorDeployReply
	err := rpcClient.Call("Deploy", arg, &reply)
	if err != nil {
		return err
	}
	log.Printf("-> %v @ %v - STATUS: %v", c.App, c.Sha, reply.Status)
	log.Println("-> " + reply.Container.String())
	return nil
}

type TeardownCommand struct {
	All        bool     `short:"a" long:"all" description:"tear down all the containers"`
	Containers []string `short:"c" long:"containers" description:"the container to tear down"`
}

func (c *TeardownCommand) Execute(args []string) error {
	overlayConfig()
	arg := SupervisorTeardownArg{}
	if c.All {
		log.Println("Supervisor Teardown all...")
		arg.All = true
	} else if c.Containers != nil && len(c.Containers) > 0 {
		log.Printf("Supervisor Teardown %v...", c.Containers)
		arg.ContainerIds = c.Containers
	} else {
		return errors.New("Please specify either all or a list of containers to teardown")
	}
	var reply SupervisorTeardownReply
	err := rpcClient.Call("Teardown", arg, &reply)
	if err != nil {
		return err
	}
	log.Printf("-> Tore down %v", reply.ContainerIds)
	log.Printf("-> %s", reply.Status)
	return nil
}

type GetCommand struct {
	Container string `short:"c" long:"container" description:"the container to get"`
}

func (c *GetCommand) Execute(args []string) error {
	overlayConfig()
	if c.Container == "" {
		return errors.New("Please specify a container to get")
	}
	arg := SupervisorGetArg{c.Container}
	var reply SupervisorGetReply
	err := rpcClient.Call("Get", arg, &reply)
	if err != nil {
		return err
	}
	log.Printf("-> Get %s : %s", c.Container, reply.Status)
	log.Printf("-> %s", reply.Container.String())
	return nil
}

type VersionCommand struct {
}

func (c *VersionCommand) Execute(args []string) error {
	overlayConfig()
	log.Println("Supervisor Version Check...")
	arg := VersionArg{}
	var reply VersionReply
	defer func() {
		if err := recover(); err != nil {
			reply.RPCVersion = "unknown"
		}
		log.Printf("-> client rpc: %s", SupervisorRPCVersion)
		log.Printf("-> server rpc: %s", reply.RPCVersion)
	}()
	return rpcClient.Call("Version", arg, &reply)
}

type AuthorizeSSHCommand struct {
	Container string `short:"c" long:"container" description:"the container to authorize"`
	User      string `short:"u" long:"user" description:"the user to authorize"`
	PublicKey string `short:"k" long:"key" description:"the user's SSH public key"`
}

func (c *AuthorizeSSHCommand) Execute(args []string) error {
	overlayConfig()
	log.Println("Authorize SSH...")
	arg := SupervisorAuthorizeSSHArg{c.Container, c.User, c.PublicKey}
	var reply SupervisorAuthorizeSSHReply
	err := rpcClient.Call("AuthorizeSSH", arg, &reply)
	if err != nil {
		return err
	}
	log.Printf("-> Authorize %s SSH for %s @ %s", reply.Status, c.User, c.Container)
	log.Printf("-> %d", reply.Port)
	return nil
}

type DeauthorizeSSHCommand struct {
	Container string `short:"c" long:"container" description:"the container to deauthorize"`
	User      string `short:"u" long:"user" description:"the user to deauthorize"`
}

func (c *DeauthorizeSSHCommand) Execute(args []string) error {
	overlayConfig()
	log.Println("Deauthorize SSH...")
	arg := SupervisorDeauthorizeSSHArg{c.Container, c.User}
	var reply SupervisorDeauthorizeSSHReply
	err := rpcClient.Call("DeauthorizeSSH", arg, &reply)
	if err != nil {
		return err
	}
	log.Printf("-> Deauthorize %s SSH for %s @ %s", reply.Status, c.User, c.Container)
	return nil
}

type ContainerMaintenanceCommand struct {
	Container   string `short:"c" long:"container" description:"the container to set maintenance for"`
	Maintenance bool   `short:"m" long:"maintenance" description:"if true, turn on maintenance mode"`
}

func (c *ContainerMaintenanceCommand) Execute(args []string) error {
	overlayConfig()
	log.Println("Container Maintenance...")
	arg := SupervisorContainerMaintenanceArg{ContainerId: c.Container, Maintenance: c.Maintenance}
	var reply SupervisorContainerMaintenanceReply
	err := rpcClient.Call("ContainerMaintenance", arg, &reply)
	if err != nil {
		return err
	}
	log.Printf("-> ContainerMaintenance %s for %s to %t", reply.Status, c.Container, c.Maintenance)
	return nil
}

type IdleCommand struct {
	Quiet bool `long:"quiet" description:"if true, quiet the output"`
}

func (c *IdleCommand) Execute(args []string) error {
	if !c.Quiet {
		log.Println("Idle ...")
	}
	arg := SupervisorIdleArg{}
	var reply SupervisorIdleReply
	err := rpcClient.Call("Idle", arg, &reply)
	if err != nil {
		return err
	}
	if c.Quiet {
		fmt.Printf("%t\n", reply.Idle)
	} else {
		log.Printf("Idle %t.", reply.Idle)
	}
	return nil
}
