package proxy

import (
	"log"
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
}

func NewHTTPProxy(lAddr, rAddr string) Proxy {
	return &HTTPProxy{
		LocalAddrString:  lAddr,
		RemoteAddrString: rAddr,
	}
}

func (p *HTTPProxy) Init() error {
	u, err := url.Parse("http://"+p.RemoteAddr())
	if err != nil {
		return err
	}
	p.rProxy = httputil.NewSingleHostReverseProxy(u)
	p.server = &http.Server {
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
	p.listenWrapper()
	p.dead <- true // if we get here, that means we paniced and are dead
}

func (p *HTTPProxy) listenWrapper() {
	defer func() {
		r := recover()
		switch r.(type) {
		case HTTPProxyDeadError:
			// sqaush panic
			p.Log("die")
		default:
			panic(r) // bubble up panic if its not our special one
		}
	}()
	err := p.server.ListenAndServe()
	if err != nil {
		panic(err)
	}
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
		panic(HTTPProxyDeadError{}) // we're supposed to die
	}
	r.Host = p.RemoteAddr()
	p.rProxy.ServeHTTP(w, r)
}

func (p *HTTPProxy) Log(format string, args ...interface{}) {
	log.Printf("["+p.LocalAddr()+"][HTTP] "+format, args...)
}
