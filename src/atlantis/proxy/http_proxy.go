package proxy

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type HTTPProxyDeadError struct{}

func (e HTTPProxyDeadError) Error() string {
	return ""
}

type HTTPProxy struct {
	LocalAddrString  string `json:"LocalAddr"`
	RemoteAddrString string `json:"RemoteAddr"`
	server           *http.Server
	rProxy           *httputil.ReverseProxy
	die              bool
	dead             chan bool
	listener         net.Listener
}

func NewHTTPProxy(lAddr, rAddr string) Proxy {
	return &HTTPProxy{
		LocalAddrString:  lAddr,
		RemoteAddrString: rAddr,
	}
}

func (p *HTTPProxy) Init() error {
	u, err := url.Parse("http://" + p.RemoteAddr())
	if err != nil {
		return err
	}
	p.rProxy = httputil.NewSingleHostReverseProxy(u)
	p.listener, err = net.Listen("tcp", p.LocalAddr())
	if err != nil {
		return err
	}
	p.server = &http.Server{
		Handler:        p,
		Addr:           p.LocalAddr(),
		ReadTimeout:    120 * time.Second,
		WriteTimeout:   120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return nil
}

func (p *HTTPProxy) LocalAddr() string {
	return p.LocalAddrString
}

func (p *HTTPProxy) RemoteAddr() string {
	return p.RemoteAddrString
}

func (p *HTTPProxy) Listen() {
	p.Log("proxying to %s", p.RemoteAddr())
	p.dead = make(chan bool)
	p.server.Serve(p.listener)
	p.dead <- true // if we get here, that means we paniced and are dead
}

func (p *HTTPProxy) Die() {
	p.die = true
	// fake request to trigger die
	if resp, err := http.Get("http://" + p.LocalAddr()); err == nil {
		resp.Body.Close()
	}
	<-p.dead
}

func (p *HTTPProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.die {
		defer p.listener.Close()
	}
	r.Host = p.RemoteAddr()
	p.rProxy.ServeHTTP(w, r)
}

func (p *HTTPProxy) Log(format string, args ...interface{}) {
	log.Printf("["+p.LocalAddr()+"][HTTP] "+format, args...)
}
