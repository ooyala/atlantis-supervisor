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

type UpdateIPGroupExecutor struct {
	arg   SupervisorUpdateIPGroupArg
	reply *SupervisorUpdateIPGroupReply
}

func (e *UpdateIPGroupExecutor) Request() interface{} {
	return e.arg
}

func (e *UpdateIPGroupExecutor) Result() interface{} {
	return e.reply
}

func (e *UpdateIPGroupExecutor) Description() string {
	return fmt.Sprintf("%s -> %v", e.arg.Name, e.arg.IPs)
}

func (e *UpdateIPGroupExecutor) Authorize() error {
	return nil
}

func (e *UpdateIPGroupExecutor) Execute(t *Task) error {
	if e.arg.Name == "" {
		return errors.New("Please specify a Name.")
	}
	if e.arg.IPs == nil {
		return errors.New("Please specify a list of IPs.")
	}
	err := containers.NetworkSecurity.UpdateIPGroup(e.arg.Name, e.arg.IPs)
	if err != nil {
		e.reply.Status = StatusOk
	} else {
		e.reply.Status = StatusError
	}
	return err
}

func (ih *Supervisor) UpdateIPGroup(arg SupervisorUpdateIPGroupArg, reply *SupervisorUpdateIPGroupReply) error {
	return NewTask("UpdateIPGroup", &UpdateIPGroupExecutor{arg, reply}).Run()
}

type DeleteIPGroupExecutor struct {
	arg   SupervisorDeleteIPGroupArg
	reply *SupervisorDeleteIPGroupReply
}

func (e *DeleteIPGroupExecutor) Request() interface{} {
	return e.arg
}

func (e *DeleteIPGroupExecutor) Result() interface{} {
	return e.reply
}

func (e *DeleteIPGroupExecutor) Description() string {
	return fmt.Sprintf("%s", e.arg.Name)
}

func (e *DeleteIPGroupExecutor) Authorize() error {
	return nil
}

func (e *DeleteIPGroupExecutor) Execute(t *Task) error {
	if e.arg.Name == "" {
		return errors.New("Please specify a Name.")
	}
	err := containers.NetworkSecurity.DeleteIPGroup(e.arg.Name)
	if err != nil {
		e.reply.Status = StatusOk
	} else {
		e.reply.Status = StatusError
	}
	return err
}

func (ih *Supervisor) DeleteIPGroup(arg SupervisorDeleteIPGroupArg, reply *SupervisorDeleteIPGroupReply) error {
	return NewTask("DeleteIPGroup", &DeleteIPGroupExecutor{arg, reply}).Run()
}
