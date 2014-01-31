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

type ContainerMaintenanceExecutor struct {
	arg   SupervisorContainerMaintenanceArg
	reply *SupervisorContainerMaintenanceReply
}

func (e *ContainerMaintenanceExecutor) Request() interface{} {
	return e.arg
}

func (e *ContainerMaintenanceExecutor) Result() interface{} {
	return e.reply
}

func (e *ContainerMaintenanceExecutor) Description() string {
	return fmt.Sprintf("%s : %t", e.arg.ContainerID, e.arg.Maintenance)
}

func (e *ContainerMaintenanceExecutor) Authorize() error {
	return nil
}

func (e *ContainerMaintenanceExecutor) Execute(t *Task) error {
	if e.arg.ContainerID == "" {
		return errors.New("Please specify a container id.")
	}
	cont := containers.Get(e.arg.ContainerID)
	if cont == nil {
		e.reply.Status = StatusError
		return errors.New("Unknown Container.")
	}
	if err := containers.SetMaintenance(cont, e.arg.Maintenance); err != nil {
		e.reply.Status = StatusError
		return err
	}
	e.reply.Status = StatusOk
	return nil
}

func (ih *Supervisor) ContainerMaintenance(arg SupervisorContainerMaintenanceArg,
	reply *SupervisorContainerMaintenanceReply) error {
	return NewTask("ContainerMaintenance", &ContainerMaintenanceExecutor{arg, reply}).Run()
}

// Supervisor Idle Check
type IdleExecutor struct {
	arg   SupervisorIdleArg
	reply *SupervisorIdleReply
}

func (e *IdleExecutor) Request() interface{} {
	return e.arg
}

func (e *IdleExecutor) Result() interface{} {
	return e.reply
}

func (e *IdleExecutor) Description() string {
	return "Idle?"
}

func (e *IdleExecutor) Execute(t *Task) error {
	e.reply.Idle = Tracker.Idle(t)
	e.reply.Status = StatusOk
	return nil
}

func (e *IdleExecutor) Authorize() error {
	return nil // let anybody ask if we're idle. i dont care.
}

func (e *IdleExecutor) AllowDuringMaintenance() bool {
	return true // allow running thus during maintenance
}

func (o *Supervisor) Idle(arg SupervisorIdleArg, reply *SupervisorIdleReply) error {
	return NewTask("Idle", &IdleExecutor{arg, reply}).Run()
}
