/* Copyright 2014 Ooyala, Inc. All rights reserved.
 *
 * This file is licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

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
