package healthz

import (
	. "atlantis/common"
	. "atlantis/supervisor/constant"
	. "atlantis/supervisor/rpc/types"
	. "atlantis/supervisor/client"
	"net/http"

	"fmt"
	"time"
)

func healthzHandler(w http.ResponseWriter, r *http.Request) {

	config := &Config{"localhost", DefaultSupervisorRPCPort}
	rpcClient := NewRPCClientWithConfig(config, "Supervisor", SupervisorRPCVersion, false)
	arg := SupervisorHealthCheckArg{}
	var reply SupervisorHealthCheckReply
	err := rpcClient.CallWithTimeout("HealthCheck", arg, &reply, 5)

	if err != nil {
		// not really sure what to do with error

		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(reply.Status))

}


func Run(port uint16) {
	http.HandleFunc("/healthz", healthzHandler)

	for {
		http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil)
		// logger.Errorf("[status server] %s", server.ListenAndServe())
		time.Sleep(1 * time.Second)
	}
}
