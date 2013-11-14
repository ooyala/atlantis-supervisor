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
	return fmt.Sprintf("%s : %t", e.arg.ContainerId, e.arg.Maintenance)
}

func (e *ContainerMaintenanceExecutor) Authorize() error {
	return nil
}

func (e *ContainerMaintenanceExecutor) Execute(t *Task) error {
	if e.arg.ContainerId == "" {
		return errors.New("Please specify a container id.")
	}
	cont := containers.Get(e.arg.ContainerId)
	if cont == nil {
		e.reply.Status = StatusError
		return errors.New("Unknown Container.")
	}
	castedCont := containers.Container(*cont)
	if err := (&castedCont).SetMaintenance(e.arg.Maintenance); err != nil {
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
