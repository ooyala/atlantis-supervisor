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

package rpc

import (
	. "atlantis/common"
	"atlantis/supervisor/containers"
	. "atlantis/supervisor/rpc/types"
	"errors"
	"fmt"
)

// Deploys an app+sha to the given container id using the given service dependencies (comes from arg.Manifest)
type DeployExecutor struct {
	arg   SupervisorDeployArg
	reply *SupervisorDeployReply
}

func (e *DeployExecutor) Request() interface{} {
	return e.arg
}

func (e *DeployExecutor) Result() interface{} {
	return e.reply
}

func (e *DeployExecutor) Description() string {
	if e.arg.Manifest == nil {
		return fmt.Sprintf("%s @ %s in %s on %s -> %s with cpu ? and mem ?", e.arg.App, e.arg.Sha, e.arg.Env,
			e.arg.Host, e.arg.ContainerID)
	}
	return fmt.Sprintf("%s @ %s in %s on %s -> %s with cpu %d and mem %d", e.arg.App, e.arg.Sha, e.arg.Env,
		e.arg.Host, e.arg.ContainerID, e.arg.Manifest.CPUShares, e.arg.Manifest.MemoryLimit)
}

func (e *DeployExecutor) Authorize() error {
	return nil
}

func (e *DeployExecutor) Execute(t *Task) error {
	if e.arg.App == "" {
		return errors.New("Please specify an app.")
	}
	if e.arg.Sha == "" {
		return errors.New("Please specify a sha.")
	}
	if e.arg.ContainerID == "" {
		return errors.New("Please specify a container id.")
	}
	if e.arg.Manifest == nil {
		return errors.New("Please specify a manifest.")
	}
	if e.arg.Manifest.CPUShares == 0 {
		return errors.New("Please specify a number of CPU shares.")
	}
	if e.arg.Manifest.MemoryLimit == 0 {
		return errors.New("Please specify a memory limit.")
	}
	cont, err := containers.Reserve(e.arg.ContainerID, e.arg.Manifest)
	if err != nil {
		t.Log("-> Error reserving container: %v", err)
		return err
	}
	err = cont.Deploy(e.arg.Host, e.arg.App, e.arg.Sha, e.arg.Env)
	if err != nil {
		cont.Teardown()
		return err
	}
	e.reply.Status = StatusOk
	e.reply.Container = &cont.Container
	return nil
}

func (ih *Supervisor) Deploy(arg SupervisorDeployArg, reply *SupervisorDeployReply) error {
	return NewTask("Deploy", &DeployExecutor{arg, reply}).Run()
}

// Teardown an already deployed container. Will return status "OK" if the container iff was found.
type TeardownExecutor struct {
	arg   SupervisorTeardownArg
	reply *SupervisorTeardownReply
}

func (e *TeardownExecutor) Request() interface{} {
	return e.arg
}

func (e *TeardownExecutor) Result() interface{} {
	return e.reply
}

func (e *TeardownExecutor) Description() string {
	return fmt.Sprintf("%v, all: %t", e.arg.ContainerIDs, e.arg.All)
}

func (e *TeardownExecutor) Authorize() error {
	return nil
}

func (e *TeardownExecutor) Execute(t *Task) error {
	if e.arg.ContainerIDs == nil && e.arg.All == false {
		return errors.New("Please specify container ids or all.")
	}
	var containerIDs []string
	if e.arg.All {
		t.Log("All requested.")
		conts, _ := containers.List()
		containerIDs = make([]string, len(conts))
		i := 0
		for id, _ := range conts {
			t.Log("-> found container %s", id)
			containerIDs[i] = id
			i++
		}
	} else {
		containerIDs = e.arg.ContainerIDs
	}
	e.reply.ContainerIDs = []string{}
	for _, containerID := range containerIDs {
		if !containers.Teardown(containerID) {
			t.Log("-> no such container: %s", containerID)
			e.reply.Status += "no such container: " + containerID + "\n"
		} else {
			e.reply.ContainerIDs = append(e.reply.ContainerIDs, containerID)
		}
	}
	if e.reply.Status == "" {
		e.reply.Status = StatusOk
	}
	return nil
}

func (ih *Supervisor) Teardown(arg SupervisorTeardownArg, reply *SupervisorTeardownReply) error {
	return NewTask("Teardown", &TeardownExecutor{arg, reply}).Run()
}
