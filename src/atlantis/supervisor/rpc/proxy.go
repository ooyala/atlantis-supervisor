package rpc

import (
	. "atlantis/common"
	"atlantis/supervisor/containers"
	. "atlantis/supervisor/rpc/types"
	"errors"
	"fmt"
)

type ConfigureProxyExecutor struct {
	arg   SupervisorConfigureProxyArg
	reply *SupervisorConfigureProxyReply
}

func (e *ConfigureProxyExecutor) Request() interface{} {
	return e.arg
}

func (e *ConfigureProxyExecutor) Result() interface{} {
	return e.reply
}

func (e *ConfigureProxyExecutor) Description() string {
	return "ConfigureProxy"
}

func (e *ConfigureProxyExecutor) Authorize() error {
	return nil
}

func (e *ConfigureProxyExecutor) Execute(t *Task) error {
	if e.arg.ProxyConfig == nil {
		return errors.New("Please specify a config.")
	}
	t.Log("New Config: %v", e.arg.ProxyConfig)
	err := containers.ConfigureProxy(e.arg.ProxyConfig)
	if err != nil {
		e.reply.Status = StatusError
		return err
	}
	e.reply.Status = StatusOk
	return nil
}

func (ih *Supervisor) ConfigureProxy(arg SupervisorConfigureProxyArg, reply *SupervisorConfigureProxyReply) error {
	return NewTask("ConfigureProxy", &ConfigureProxyExecutor{arg, reply}).Run()
}

type UpdateProxyExecutor struct {
	arg   SupervisorUpdateProxyArg
	reply *SupervisorUpdateProxyReply
}

func (e *UpdateProxyExecutor) Request() interface{} {
	return e.arg
}

func (e *UpdateProxyExecutor) Result() interface{} {
	return e.reply
}

func (e *UpdateProxyExecutor) Description() string {
	return fmt.Sprintf("%s on %s", e.arg.Sha, e.arg.Host)
}

func (e *UpdateProxyExecutor) Authorize() error {
	return nil
}

func (e *UpdateProxyExecutor) Execute(t *Task) error {
	if e.arg.Sha == "" {
		return errors.New("Please specify a sha.")
	}
	err := containers.UpdateProxy(e.arg.Host, e.arg.Sha)
	if err != nil {
		e.reply.Status = StatusError
		return err
	}
	e.reply.Status = StatusOk
	if containers.Proxy == nil {
		e.reply.Proxy = nil
	} else {
		e.reply.Proxy = &containers.Proxy.ProxyContainer
	}
	return nil
}

func (ih *Supervisor) UpdateProxy(arg SupervisorUpdateProxyArg, reply *SupervisorUpdateProxyReply) error {
	return NewTask("UpdateProxy", &UpdateProxyExecutor{arg, reply}).Run()
}

type GetProxyExecutor struct {
	arg   SupervisorGetProxyArg
	reply *SupervisorGetProxyReply
}

func (e *GetProxyExecutor) Request() interface{} {
	return e.arg
}

func (e *GetProxyExecutor) Result() interface{} {
	return e.reply
}

func (e *GetProxyExecutor) Description() string {
	return "GetProxy"
}

func (e *GetProxyExecutor) Authorize() error {
	return nil
}

func (e *GetProxyExecutor) Execute(t *Task) error {
	if containers.Proxy == nil {
		e.reply.Proxy = nil
	} else {
		e.reply.Proxy = &containers.Proxy.ProxyContainer
	}
	return nil
}

func (ih *Supervisor) GetProxy(arg SupervisorGetProxyArg, reply *SupervisorGetProxyReply) error {
	return NewTask("GetProxy", &GetProxyExecutor{arg, reply}).Run()
}
