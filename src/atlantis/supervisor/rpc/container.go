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

// List all the deployed containers and available ports
type ListExecutor struct {
	arg   SupervisorListArg
	reply *SupervisorListReply
}

func (e *ListExecutor) Request() interface{} {
	return e.arg
}

func (e *ListExecutor) Result() interface{} {
	return e.reply
}

func (e *ListExecutor) Description() string {
	return "List"
}

func (e *ListExecutor) Authorize() error {
	return nil
}

func (e *ListExecutor) Execute(t *Task) error {
	e.reply.Containers, e.reply.UnusedPorts = containers.List()
	return nil
}

func (ih *Supervisor) List(arg SupervisorListArg, reply *SupervisorListReply) error {
	return NewTask("List", &ListExecutor{arg, reply}).Run()
}

type GetExecutor struct {
	arg   SupervisorGetArg
	reply *SupervisorGetReply
}

func (e *GetExecutor) Request() interface{} {
	return e.arg
}

func (e *GetExecutor) Result() interface{} {
	return e.reply
}

func (e *GetExecutor) Description() string {
	return fmt.Sprintf("%s", e.arg.ContainerID)
}

func (e *GetExecutor) Authorize() error {
	return nil
}

func (e *GetExecutor) Execute(t *Task) (err error) {
	e.reply.Container = containers.Get(e.arg.ContainerID)
	if e.reply.Container == nil {
		e.reply.Status = StatusError
		err = errors.New("Unknown Container.")
	} else {
		e.reply.Status = StatusOk
	}
	return
}

func (ih *Supervisor) Get(arg SupervisorGetArg, reply *SupervisorGetReply) (err error) {
	return NewTask("Get", &GetExecutor{arg, reply}).Run()
}
