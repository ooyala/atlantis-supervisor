package proxy

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/cespare/go-apachelog"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	NotProxyingError = iota
	AlreadyProxyingError
)

type MultiProxyError struct {
	Msg  string
	Code int
}

func (e *MultiProxyError) Error() string {
	return e.Msg
}

func NewNotProxyingError(lAddr string) *MultiProxyError {
	return &MultiProxyError{Msg: "Not Proxying " + lAddr, Code: NotProxyingError}
}

func NewAlreadyProxyingError(lAddr, rAddr string) *MultiProxyError {
	return &MultiProxyError{Msg: "Already Proxying " + lAddr + " to " + rAddr, Code: NotProxyingError}
}

type MultiProxy struct {
	sync.Mutex
	SaveFile           string
	ConfigAddr         string
	DefaultNumHandlers int
	DefaultMaxPending  int
	ProxyMap           map[string]*Proxy // local address -> proxy
}

func NewMultiProxy(saveFile, cAddr string, numHandlers, maxPending int) *MultiProxy {
	return &MultiProxy{
		Mutex:              sync.Mutex{},
		SaveFile:           saveFile,
		ConfigAddr:         cAddr,
		DefaultNumHandlers: numHandlers,
		DefaultMaxPending:  maxPending,
		ProxyMap:           map[string]*Proxy{},
	}
}

func (p *MultiProxy) AddProxy(localAddr, remoteAddr string, numHandlers, maxPending int) error {
	p.Lock()
	defer p.Unlock()
	if proxy, ok := p.ProxyMap[localAddr]; ok && proxy != nil {
		return NewAlreadyProxyingError(localAddr, proxy.RemoteAddrString)
	}
	if numHandlers <= 0 {
		numHandlers = p.DefaultNumHandlers
	}
	if maxPending <= 0 {
		maxPending = p.DefaultMaxPending
	}
	proxy, err := NewProxy(localAddr, remoteAddr, numHandlers, maxPending)
	if err != nil {
		return err
	}
	p.ProxyMap[localAddr] = proxy
	go proxy.Listen()
	return nil
}

func (p *MultiProxy) RemoveProxy(localAddr string) error {
	p.Lock()
	defer p.Unlock()
	if proxy, ok := p.ProxyMap[localAddr]; !ok || proxy == nil {
		return NewNotProxyingError(localAddr)
	} else {
		proxy.die = true
		// fake request to trigger die
		if resp, err := http.Get("http://" + localAddr); err == nil {
			resp.Body.Close()
		}
		<-proxy.dead
		delete(p.ProxyMap, localAddr)
	}
	return nil
}

func (p *MultiProxy) Listen() error {
	p.load()
	// listen for config changes
	gmux := mux.NewRouter() // Use gorilla mux for APIs to make things easier
	gmux.HandleFunc("/proxy/{local}/{remote}", p.AddProxyHandler).Methods("PUT")
	gmux.HandleFunc("/proxy/{local}/{remote}", p.RemoveProxyHandler).Methods("DELETE")
	gmux.HandleFunc("/proxy/{local}", p.RemoveProxyHandler).Methods("DELETE")
	gmux.HandleFunc("/config", p.GetConfigHandler).Methods("GET")

	server := &http.Server{Addr: p.ConfigAddr, Handler: apachelog.NewHandler(gmux, os.Stderr)}
	log.Println("[CONFIG] listening on " + p.ConfigAddr)
	log.Fatal(server.ListenAndServe())
	return nil
}

func (p *MultiProxy) AddProxyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	local := sanitizeAddr(vars["local"])
	remote := sanitizeAddr(vars["remote"])
	numHandlers, _ := strconv.Atoi(r.FormValue("numHandlers"))
	maxPending, _ := strconv.Atoi(r.FormValue("maxPending"))
	if err := p.AddProxy(local, remote, numHandlers, maxPending); err != nil {
		switch err.(type) {
		case *MultiProxyError:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	p.save()
	log.Println("[CONFIG] added %s -> %s", local, remote)
	fmt.Fprintf(w, "added %s -> %s", local, remote)
}

func (p *MultiProxy) RemoveProxyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	local := sanitizeAddr(vars["local"])
	if err := p.RemoveProxy(local); err != nil {
		switch err.(type) {
		case *MultiProxyError:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	p.save()
	log.Println("[CONFIG] removed %s", local)
	fmt.Fprintf(w, "removed %s", local)
}

func (p *MultiProxy) GetConfigHandler(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	enc.Encode(p.ProxyMap)
}

func sanitizeAddr(addr string) string {
	if strings.Index(addr, ":") < 0 {
		return addr + ":80"
	}
	return addr
}

func (p *MultiProxy) save() {
	p.Lock()
	gob.Register(p)
	fo, err := os.Create(p.SaveFile)
	if err != nil {
		log.Printf("[CONFIG] could not save %s: %s", p.SaveFile, err)
		return
	}
	defer fo.Close()
	w := bufio.NewWriter(fo)
	e := gob.NewEncoder(w)
	e.Encode(p)
	w.Flush()
	p.Unlock()
}

func (p *MultiProxy) load() {
	p.Lock()
	fi, err := os.Open(p.SaveFile)
	if err != nil {
		log.Printf("[CONFIG] could not retrieve %s: %s", p.SaveFile, err)
	}
	r := bufio.NewReader(fi)
	d := gob.NewDecoder(r)
	d.Decode(p)
	for _, proxy := range p.ProxyMap {
		proxy.Listen()
	}
	p.Unlock()
}
