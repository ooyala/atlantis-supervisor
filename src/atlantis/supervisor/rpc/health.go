package rpc

import (
	. "atlantis/common"
	. "atlantis/supervisor/constant"
	"atlantis/supervisor/containers"
	. "atlantis/supervisor/rpc/types"
)

// Check the health of Supervisor. Supervisor will return some useful stats as well.
type HealthCheckExecutor struct {
	arg   SupervisorHealthCheckArg
	reply *SupervisorHealthCheckReply
}

func (e *HealthCheckExecutor) Request() interface{} {
	return e.arg
}

func (e *HealthCheckExecutor) Result() interface{} {
	return e.reply
}

func (e *HealthCheckExecutor) Description() string {
	return "HealthCheck"
}

func (e *HealthCheckExecutor) Authorize() error {
	return nil
}

func (e *HealthCheckExecutor) AllowDuringMaintenance() bool {
	return true // allow running thus during maintenance
}

func (e *HealthCheckExecutor) Execute(t *Task) error {
	e.reply.Region = Region
	e.reply.Zone = Zone
	e.reply.Containers, e.reply.CPUShares, e.reply.Memory = containers.Nums()
	if Tracker.UnderMaintenance() {
		e.reply.Status = StatusMaintenance
	} else if e.reply.Containers.Free == 0 || e.reply.Memory.Free == 0 || e.reply.CPUShares.Free == 0 {
		e.reply.Status = StatusFull
	} else {
		e.reply.Status = StatusOk
	}
	t.Log("-> region: %s, zone: %s", e.reply.Region, e.reply.Zone)
	t.Log("-> containers: %d total, %d used, %d free", e.reply.Containers.Total,
		e.reply.Containers.Used, e.reply.Containers.Free)
	t.Log("-> cpu shares: %d total, %d used, %d free", e.reply.CPUShares.Total,
		e.reply.CPUShares.Used, e.reply.CPUShares.Free)
	t.Log("-> memory: %d MB total, %d MB used, %d MB free", e.reply.Memory.Total,
		e.reply.Memory.Used, e.reply.Memory.Free)
	t.Log("-> status: %s", e.reply.Status)
	return nil
}

func (ih *Supervisor) HealthCheck(arg SupervisorHealthCheckArg, reply *SupervisorHealthCheckReply) (err error) {
	return NewTask("HealthCheck", &HealthCheckExecutor{arg, reply}).Run()
}
