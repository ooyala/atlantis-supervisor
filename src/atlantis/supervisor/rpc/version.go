package rpc

import (
	. "atlantis/common"
	. "atlantis/supervisor/constant"
)

// Return the current RPC Version
type VersionExecutor struct {
	arg   VersionArg
	reply *VersionReply
}

func (e *VersionExecutor) Request() interface{} {
	return e.arg
}

func (e *VersionExecutor) Result() interface{} {
	return e.reply
}

func (e *VersionExecutor) Description() string {
	return "Version"
}

func (e *VersionExecutor) Authorize() error {
	return nil
}

func (e *VersionExecutor) AllowDuringMaintenance() bool {
	return true // allow running thus during maintenance
}

func (e *VersionExecutor) Execute(t *Task) error {
	e.reply.RPCVersion = SupervisorRPCVersion
	return nil
}

func (ih *Supervisor) Version(arg VersionArg, reply *VersionReply) error {
	return NewTask("Version", &VersionExecutor{arg, reply}).Run()
}
