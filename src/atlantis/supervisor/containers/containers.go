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
	"atlantis/supervisor/containers/serialize"
	"atlantis/supervisor/docker"
	"atlantis/supervisor/netsec"
	"atlantis/supervisor/rpc/types"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	ContainersFile      = "containers"
	PortsFile           = "ports"
	NetworkSecurityFile = "netsec"
)

type ReserveReq struct {
	id       string
	manifest *types.Manifest
	respChan chan *ReserveResp
}

type ReserveResp struct {
	container *Container
	err       error
}

type TeardownReq struct {
	id       string
	respChan chan bool
}

type GetReq struct {
	id       string
	respChan chan *types.Container
}

type ListResp struct {
	containers map[string]*types.Container
	ports      []uint16
}

type NumsResp struct {
	Containers *types.ResourceStats
	CPUShares  *types.ResourceStats
	Memory     *types.ResourceStats
}

var (
	NetworkSecurity   *netsec.NetworkSecurity
	EnableNetsec      bool
	NumContainers     uint16 // for maximum efficiency, should = CPUShares
	NumSecondaryPorts uint16
	MinPort           uint16
	CPUShares         uint // relative
	MemoryLimit       uint // actual MB
	reserveChan       chan *ReserveReq
	teardownChan      chan *TeardownReq
	getChan           chan *GetReq
	listChan          chan chan *ListResp
	numsChan          chan chan *NumsResp
	dieChan           chan bool
	containers        map[string]*Container // not for direct access. must go through containerManager.
	ports             []uint16              // not for direct access. must go through containerManager.
	usedMemoryLimit   uint                  // not for direct access. must go through containerManager.
	usedCPUShares     uint                  // not for direct access. must go through containerManager.
)

// Initialize everything needed to use containers
func Init(registry, saveDir string, numContainers, numSecondaryPorts, minPort uint16, cpu, memory uint) error {
	if err := serialize.Init(saveDir); err != nil {
		return err
	}
	NumContainers = numContainers
	NumSecondaryPorts = numSecondaryPorts
	MinPort = minPort
	CPUShares = cpu
	MemoryLimit = memory
	if uint64(MinPort)+(uint64(NumSecondaryPorts)+2)*uint64(NumContainers)-1 > 65535 {
		return errors.New("Invalid Config. MinPort+(NumSecondaryPorts+2)*NumContainers-1 > 65535")
	}
	if uint(NumContainers) != CPUShares {
		// don't error out because technically this is ok
		log.Println("WARNING: for maximum efficiency please set num_containers = cpu_shares")
	}
	reserveChan = make(chan *ReserveReq)
	teardownChan = make(chan *TeardownReq)
	getChan = make(chan *GetReq)
	listChan = make(chan chan *ListResp)
	numsChan = make(chan chan *NumsResp)
	dieChan = make(chan bool)
	if err := docker.Init(registry); err != nil {
		return err
	}
	go containerManager()
	return nil
}

// Reserve a container
func Reserve(id string, manifest *types.Manifest) (*Container, error) {
	respChan := make(chan *ReserveResp)
	req := &ReserveReq{id, manifest, respChan}
	reserveChan <- req
	resp := <-respChan
	close(respChan)
	return resp.container, resp.err
}

// Teardown a container
func Teardown(id string) bool {
	respChan := make(chan bool)
	req := &TeardownReq{id, respChan}
	teardownChan <- req
	resp := <-respChan
	close(respChan)
	return resp
}

func Get(id string) *types.Container {
	respChan := make(chan *types.Container)
	req := &GetReq{id, respChan}
	getChan <- req
	resp := <-respChan
	close(respChan)
	return resp
}

// List all containers and free ports
func List() (map[string]*types.Container, []uint16) {
	respChan := make(chan *ListResp)
	listChan <- respChan
	resp := <-respChan
	close(respChan)
	return resp.containers, resp.ports
}

// Return the number of total, used, and free containers
func Nums() (containers *types.ResourceStats, cpu *types.ResourceStats, memory *types.ResourceStats) {
	respChan := make(chan *NumsResp)
	numsChan <- respChan
	resp := <-respChan
	close(respChan)
	return resp.Containers, resp.CPUShares, resp.Memory
}

func reserve(req *ReserveReq) {
	resp := &ReserveResp{}
	if len(containers) >= int(NumContainers) { // check if there are enough containers
		resp.err = errors.New("No free containers to reserve.")
	} else if containers[req.id] != nil {
		resp.err = errors.New("The ID (" + req.id + ") is in use.")
	} else if req.manifest.CPUShares+usedCPUShares > CPUShares { // check cpu
		resp.err = errors.New(fmt.Sprintf("Not enough CPU Shares to reserve. (%d requested, %d available)",
			req.manifest.CPUShares, CPUShares-usedCPUShares))
	} else if req.manifest.MemoryLimit+usedMemoryLimit > MemoryLimit { // check memory
		resp.err = errors.New(fmt.Sprintf("Not enough Memory to reserve. (%d requested, %d available)",
			req.manifest.MemoryLimit, MemoryLimit-usedMemoryLimit))
	} else {
		port := ports[0]
		ports = ports[1:]
		secondaryPorts := make([]uint16, NumSecondaryPorts)
		for i := uint16(0); i < NumSecondaryPorts; i++ {
			secondaryPorts[i] = MinPort + (NumContainers * (i + 2)) + port
		}
		containers[req.id] = &Container{Container: types.Container{ID: req.id, PrimaryPort: MinPort + port,
			SSHPort: MinPort + NumContainers + port, SecondaryPorts: secondaryPorts, Manifest: req.manifest}}
		resp.container = containers[req.id]
		usedMemoryLimit = usedMemoryLimit + req.manifest.MemoryLimit
		usedCPUShares = usedCPUShares + req.manifest.CPUShares
	}
	req.respChan <- resp
	return
}

func teardown(req *TeardownReq) {
	container := containers[req.id]
	if container != nil {
		NetworkSecurity.RemoveContainerSecurity(req.id)
		docker.Teardown(containers[req.id])
		ports = append(ports, containers[req.id].PrimaryPort-MinPort)
		usedMemoryLimit = usedMemoryLimit - containers[req.id].Manifest.MemoryLimit
		usedCPUShares = usedCPUShares - containers[req.id].Manifest.CPUShares
		delete(containers, req.id)
		save()
		go func() {
			// inventory() eventually calls back into the supervisor via cmk_admin -I
			<-time.After(100 * time.Millisecond)
			uploadLog(req.id)
			inventory()
		}()
		req.respChan <- true
	} else {
		req.respChan <- false
	}
}

func get(req *GetReq) {
	container, present := containers[req.id]
	if !present {
		req.respChan <- nil
	} else {
		castedContainer := container.Container
		req.respChan <- &castedContainer
	}
}

func list(respChan chan *ListResp) {
	// create copies
	portsCopy := make([]uint16, len(ports))
	for i, port := range ports {
		portsCopy[i] = MinPort + port
	}
	containersCopy := make(map[string]*types.Container, len(containers))
	for id, container := range containers {
		castedContainer := container.Container
		containersCopy[id] = &castedContainer
	}
	resp := &ListResp{containersCopy, portsCopy}
	respChan <- resp
}

func nums(respChan chan *NumsResp) {
	resp := &NumsResp{&types.ResourceStats{uint(NumContainers), uint(len(containers)),
		uint(NumContainers) - uint(len(containers))}, &types.ResourceStats{CPUShares, usedCPUShares,
		CPUShares - usedCPUShares}, &types.ResourceStats{MemoryLimit, usedMemoryLimit,
		MemoryLimit - usedMemoryLimit}}
	respChan <- resp
}

func containerManager() {
	if err := serialize.RetrieveObject(ContainersFile, &containers); err != nil || containers == nil {
		containers = map[string]*Container{}
		log.Printf("-> using default container map: %+v", containers)
	}
	if err := serialize.RetrieveObject(PortsFile, &ports); err != nil || ports == nil {
		ports = make([]uint16, NumContainers)
		for i := uint16(0); i < NumContainers; i++ {
			ports[i] = i
		}
		log.Printf("-> using default port list: %+v", ports)
	}
	var ns netsec.NetworkSecurity
	if err := serialize.RetrieveObject(NetworkSecurityFile, ns); err != nil {
		// Enable is negated because it is "Pretend" on the inside, "Enable" on the outside.
		NetworkSecurity = netsec.New(NetworkSecurityFile, !EnableNetsec)
		log.Printf("-> using default network security (wide open)")
	} else {
		NetworkSecurity = &ns
	}
	usedCPUShares = 0
	usedMemoryLimit = 0
	for _, cont := range containers {
		usedCPUShares += cont.Manifest.CPUShares
		usedMemoryLimit += cont.Manifest.MemoryLimit
	}
	var reserveReq *ReserveReq
	var teardownReq *TeardownReq
	var getReq *GetReq
	var listRespCh chan *ListResp
	var numsRespCh chan *NumsResp
	for {
		select {
		case reserveReq = <-reserveChan:
			reserve(reserveReq)
		case teardownReq = <-teardownChan:
			teardown(teardownReq)
		case listRespCh = <-listChan:
			list(listRespCh)
		case getReq = <-getChan:
			get(getReq)
		case numsRespCh = <-numsChan:
			nums(numsRespCh)
		case <-dieChan:
			close(reserveChan)
			close(teardownChan)
			close(listChan)
			close(numsChan)
			close(dieChan)
			return
		}
	}
}

func save() {
	serialize.SaveAll(serialize.SaveDefinition{
		ContainersFile,
		containers,
	}, serialize.SaveDefinition{
		PortsFile,
		ports,
	})
}

func inventory() {
	log.Println("[CMK Inventory] Start")
	cmd := exec.Command("cmk_admin", "-I")
	output, err := cmd.Output()
	if err != nil {
		log.Println("[CMK Inventory] ERROR: " + err.Error() + "\n" + string(output))
	} else {
		log.Println("[CMK Inventory] done:\n" + string(output))
	}
}

func uploadLog(id string) {
	log.Println("[Teardown Logsync] Start")
	os.Chdir("/opt/atlantis/logsync")
	output, err := exec.Command("bash", "-c", "./run -suffix=.log -region=`my-region` -once").Output()
	if err != nil {
		log.Println("[Teardown Logsync] ERROR: " + err.Error() + "\n" + string(output))
	} else {
		log.Println("[Teardown Logsync] done:\n" + string(output))
	}
}
