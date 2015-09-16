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

package server

import (
	. "atlantis/common"
	"atlantis/crypto"
	. "atlantis/supervisor/constant"
	"atlantis/supervisor/containers"
	"atlantis/supervisor/healthz"
	"atlantis/supervisor/rpc"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/jigish/go-flags"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	SaveDir                  string  `toml:"save_dir"`
	NumContainers            uint16  `toml:"num_containers"`
	NumSecondary             uint16  `toml:"num_secondary"`
	CPUShares                uint    `toml:"cpu_shares"`
	MemoryLimit              uint    `toml:"memory_limit"`
	MinPort                  uint16  `toml:"min_port"`
	RpcAddr                  string  `toml:"rpc_addr"`
	RegistryHost             string  `toml:"registry_host"`
	ResultDuration           string  `toml:"result_duration"`
	Region                   string  `toml:"region"`
	Zone                     string  `toml:"zone"`
	MaintenanceFile          string  `toml:"maintenance_file"`
	MaintenanceCheckInterval string  `toml:"maintenance_check_interval"`
	EnableNetsec             bool    `toml:"enable_netsec"`
	Price                    float64 `toml:"price"`
}

type Opts struct {
	SaveDir                  string  `long:"save" description:"the directory to save to"`
	NumContainers            uint16  `long:"containers" description:"the # of available containers"`
	NumSecondary             uint16  `long:"secondary" description:"the # of secondary ports"`
	CPUShares                uint    `long:"cpu-shares" description:"the total # of CPU shares available"`
	MemoryLimit              uint    `long:"memory-limit" description:"the total MB of memory available"`
	MinPort                  uint16  `long:"min-port" description:"the minimum port number to use"`
	RpcAddr                  string  `long:"rpc" description:"the RPC listen addr"`
	RegistryHost             string  `long:"registry" description:"the Registry Host to talk to"`
	ResultDuration           string  `long:"result-duration" description:"How long to keep the results of an Async Command"`
	Region                   string  `long:"region" description:"the region this supervisor is in"`
	Zone                     string  `long:"zone" description:"the availability zone this supervisor is in"`
	Config                   string  `long:"config-file" default:"/etc/atlantis/supervisor/server.toml" description:"the config file to use"`
	MaintenanceFile          string  `long:"maintenance-file" description:"the maintenance file to check"`
	MaintenanceCheckInterval string  `long:"maintenance-check-interval" description:"the interval to check the maintenance file"`
	EnableNetsec             bool    `long:"enable-netsec" description:"enable network security (iptables)"`
	Price                    float64 `long:"price"`
}

var opts = &Opts{}
var config = &Config{
	SaveDir:                  DefaultSupervisorSaveDir,
	NumContainers:            DefaultSupervisorNumContainers,
	NumSecondary:             DefaultSupervisorNumSecondary,
	CPUShares:                DefaultSupervisorCPUShares,
	MemoryLimit:              DefaultSupervisorMemoryLimit,
	MinPort:                  DefaultSupervisorMinPort,
	RpcAddr:                  fmt.Sprintf(":%d", DefaultSupervisorRPCPort),
	RegistryHost:             DefaultSupervisorRegistryHost,
	ResultDuration:           DefaultResultDuration,
	Region:                   DefaultRegion,
	Zone:                     DefaultZone,
	MaintenanceFile:          DefaultMaintenanceFile,
	MaintenanceCheckInterval: DefaultMaintenanceCheckInterval,
	EnableNetsec:             false,
}

type Supervisor struct {
	parser *flags.Parser
}

// Creates and returns a new Supervisor
func New() *Supervisor {
	ih := &Supervisor{}
	ih.parser = flags.NewParser(opts, flags.Default)
	return ih
}

func (ih *Supervisor) Run() {
	ih.parser.Parse()
	log.Println("You feelin' lucky, punk?")
	log.Println("                          -- Supervisor\n")
	crypto.Init()
	overlayConfig()
	Region = config.Region
	Zone = config.Zone
	Price = config.Price
	log.Printf("Initializing Atlantis Supervisor [%s] [%s]", Region, Zone)
	handleError(containers.Init(config.RegistryHost, config.SaveDir, config.NumContainers, config.NumSecondary,
		config.MinPort, config.CPUShares, config.MemoryLimit, config.EnableNetsec))
	handleError(rpc.Init(config.RpcAddr))
	maintenanceCheckInterval, err := time.ParseDuration(config.MaintenanceCheckInterval)
	if err != nil {
		log.Fatalln(err)
	}
	go healthz.Run(8080)
	go signalListener()
	MaintenanceChecker(config.MaintenanceFile, maintenanceCheckInterval)
	rpc.Listen()
}

func handleError(err error) {
	if err != nil {
		log.Fatalln("ERROR:", err)
	}
}

func overlayConfig() {
	if opts.Config != "" {
		_, err := toml.DecodeFile(opts.Config, config)
		if err != nil {
			log.Println(err)
			// no need to panic here. we have reasonable defaults.
		}
	}
	if opts.SaveDir != "" {
		config.SaveDir = opts.SaveDir
	}
	if opts.NumContainers != 0 {
		config.NumContainers = opts.NumContainers
	}
	if opts.NumSecondary != 0 {
		config.NumSecondary = opts.NumSecondary
	}
	if opts.MinPort != 0 {
		config.MinPort = opts.MinPort
	}
	if opts.RpcAddr != "" {
		config.RpcAddr = opts.RpcAddr
	}
	if opts.RegistryHost != "" {
		config.RegistryHost = opts.RegistryHost
	}
	if opts.ResultDuration != "" {
		config.ResultDuration = opts.ResultDuration
	}
	if opts.Region != "" {
		config.Region = opts.Region
	}
	if opts.Zone != "" {
		config.Zone = opts.Zone
	}
	if opts.MaintenanceFile != "" {
		config.MaintenanceFile = opts.MaintenanceFile
	}
	if opts.MaintenanceCheckInterval != "" {
		config.MaintenanceCheckInterval = opts.MaintenanceCheckInterval
	}
	if opts.EnableNetsec {
		config.EnableNetsec = opts.EnableNetsec
	}
}

func signalListener() {
	// wait for SIGTERM
	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGTERM)
	<-termChan
	signal.Stop(termChan)
	close(termChan)

	// wait indefinitely for idle before exit - we can always kill if we *really* want supervisor to die
	log.Println("[SIGTERM] Gracefully shutting down...")
	for !Tracker.Idle(nil) {
		log.Println("[SIGTERM] -> waiting for idle")
		time.Sleep(5 * time.Second)
	}
	os.Exit(0)
}
