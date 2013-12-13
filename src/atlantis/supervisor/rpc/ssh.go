package rpc

import (
	. "atlantis/common"
	"atlantis/supervisor/containers"
	. "atlantis/supervisor/rpc/types"
	"errors"
	"fmt"
)

type AuthorizeSSHExecutor struct {
	arg   SupervisorAuthorizeSSHArg
	reply *SupervisorAuthorizeSSHReply
}

func (e *AuthorizeSSHExecutor) Request() interface{} {
	return e.arg
}

func (e *AuthorizeSSHExecutor) Result() interface{} {
	return e.reply
}

func (e *AuthorizeSSHExecutor) Description() string {
	return fmt.Sprintf("%s @ %s :\n%s", e.arg.User, e.arg.ContainerID, e.arg.PublicKey)
}

func (e *AuthorizeSSHExecutor) Authorize() error {
	return nil
}

func (e *AuthorizeSSHExecutor) Execute(t *Task) error {
	if e.arg.PublicKey == "" {
		return errors.New("Please specify an SSH public key.")
	}
	if e.arg.ContainerID == "" {
		return errors.New("Please specify a container id.")
	}
	if e.arg.User == "" {
		return errors.New("Please specify a user.")
	}
	cont := containers.Get(e.arg.ContainerID)
	if cont == nil {
		e.reply.Status = StatusError
		return errors.New("Unknown Container.")
	}
	castedCont := containers.Container(*cont)
	if err := (&castedCont).AuthorizeSSHUser(e.arg.User, e.arg.PublicKey); err != nil {
		e.reply.Status = StatusError
		return err
	}
	t.Log("[RPC][AuthorizeSSH] authorized %d", cont.SSHPort)
	e.reply.Port = cont.SSHPort
	e.reply.Status = StatusOk
	return nil
}

func (ih *Supervisor) AuthorizeSSH(arg SupervisorAuthorizeSSHArg, reply *SupervisorAuthorizeSSHReply) error {
	return NewTask("AuthorizeSSH", &AuthorizeSSHExecutor{arg, reply}).Run()
}

type DeauthorizeSSHExecutor struct {
	arg   SupervisorDeauthorizeSSHArg
	reply *SupervisorDeauthorizeSSHReply
}

func (e *DeauthorizeSSHExecutor) Request() interface{} {
	return e.arg
}

func (e *DeauthorizeSSHExecutor) Result() interface{} {
	return e.reply
}

func (e *DeauthorizeSSHExecutor) Description() string {
	return fmt.Sprintf("%s @ %s", e.arg.User, e.arg.ContainerID)
}

func (e *DeauthorizeSSHExecutor) Authorize() error {
	return nil
}

func (e *DeauthorizeSSHExecutor) Execute(t *Task) error {
	if e.arg.ContainerID == "" {
		return errors.New("Please specify a container id.")
	}
	if e.arg.User == "" {
		return errors.New("Please specify a user.")
	}
	cont := containers.Get(e.arg.ContainerID)
	if cont == nil {
		e.reply.Status = StatusError
		return errors.New("Unknown Container.")
	}
	castedCont := containers.Container(*cont)
	if err := (&castedCont).DeauthorizeSSHUser(e.arg.User); err != nil {
		e.reply.Status = StatusError
		return err
	}
	e.reply.Status = StatusOk
	return nil
}

func (ih *Supervisor) DeauthorizeSSH(arg SupervisorDeauthorizeSSHArg, reply *SupervisorDeauthorizeSSHReply) error {
	return NewTask("DeauthorizeSSH", &DeauthorizeSSHExecutor{arg, reply}).Run()
}
