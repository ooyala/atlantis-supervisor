package client

import (
	"atlantis/common"
	. "atlantis/supervisor/constant"
)

func NewSupervisorRPCClient(hostAndPort string) *common.RPCClient {
	return common.NewRPCClient(hostAndPort, "Supervisor", SupervisorRPCVersion, false)
}
