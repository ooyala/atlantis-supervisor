package rpc

import (
	"atlantis/common"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"time"
)

type Supervisor bool

var (
	lAddr string
	l     net.Listener
)

func Init(listenAddr string) error {
	// init rpc stuff here
	common.Tracker.ResultDuration = 30 * time.Minute
	lAddr = listenAddr
	supervisor := new(Supervisor)
	rpc.Register(supervisor)
	rpc.HandleHTTP()
	var e error
	l, e = net.Listen("tcp", listenAddr)
	if e != nil {
		return e
	}
	return nil
}

func Listen() {
	if l == nil {
		panic("Not Initialized.")
	}
	log.Println("[RPC] Listening on", lAddr)
	log.Fatal(http.Serve(l, nil).Error())
}
