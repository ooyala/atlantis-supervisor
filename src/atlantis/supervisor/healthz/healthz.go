package healthz

import (
	. "atlantis/common"
	. "atlantis/supervisor/constant"
	. "atlantis/supervisor/rpc/types"
	. "atlantis/supervisor/client"
	"fmt"
	"log"
	"net/http"
	"time"
)

func healthzHandler(w http.ResponseWriter, r *http.Request) {

	config := &Config{"localhost", DefaultSupervisorRPCPort}
	rpcClient := NewRPCClientWithConfig(config, "Supervisor", SupervisorRPCVersion, false)
	arg := SupervisorHealthCheckArg{}
	var reply SupervisorHealthCheckReply
	err := rpcClient.CallWithTimeout("HealthCheck", arg, &reply, 5)

	if err != nil {
		log.Println("ERROR: ", err)
		fmt.Fprintf(w, "CRITICAL")
		return
	}

	fmt.Fprintf(w, reply.Status)

}


func Run(port uint16) {
	http.HandleFunc("/healthz", healthzHandler)

	for {
		log.Println("[healthz server] %s", http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil))
		time.Sleep(1 * time.Second)
	}
}
