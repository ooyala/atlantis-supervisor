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
	return fmt.Sprintf("%s", e.arg.ContainerId)
}

func (e *GetExecutor) Authorize() error {
	return nil
}

func (e *GetExecutor) Execute(t *Task) (err error) {
	e.reply.Container = containers.Get(e.arg.ContainerId)
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
