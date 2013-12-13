package proxy

import (
	"io"
	"log"
	"net"
	"net/http"
)

type TCPProxy struct {
	NumHandlers      int
	MaxPending       int
	LocalAddrString  string `json:"LocalAddr"`
	localAddr        *net.TCPAddr
	RemoteAddrString string `json:"RemoteAddr"`
	remoteAddr       *net.TCPAddr
	listener         *net.TCPListener
	dead             chan bool
	die              bool
}

func NewTCPProxy(lAddr, rAddr string, numHandlers, maxPending int) Proxy {
	return &TCPProxy{
		NumHandlers:      numHandlers,
		MaxPending:       maxPending,
		LocalAddrString:  lAddr,
		RemoteAddrString: rAddr,
	}
}

func (p *TCPProxy) Init() error {
	localAddr, err := net.ResolveTCPAddr("tcp", p.LocalAddr())
	if err != nil {
		return err
	}
	remoteAddr, err := net.ResolveTCPAddr("tcp", p.RemoteAddr())
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", localAddr)
	if err != nil {
		return err
	}
	p.localAddr = localAddr
	p.remoteAddr = remoteAddr
	p.listener = listener
	return nil
}

func (p *TCPProxy) LocalAddr() string {
	return p.LocalAddrString
}

func (p *TCPProxy) RemoteAddr() string {
	return p.RemoteAddrString
}

func (p *TCPProxy) Listen() {
	p.Log("proxying to %s", p.RemoteAddr())
	p.dead = make(chan bool)

	pending := make(chan *net.TCPConn, p.MaxPending)
	die := make(chan bool, p.NumHandlers)

	for i := 0; i < p.NumHandlers; i++ {
		go p.handleConn(i, pending, die)
	}

	for {
		if p.die {
			p.Log("die")
			if err := p.listener.Close(); err != nil {
				p.Log("ERROR: " + err.Error())
			}
			for i := 0; i < p.NumHandlers; i++ {
				die <- true
			}
			p.dead <- true
			return
		}
		conn, err := p.listener.AcceptTCP()
		if err != nil {
			p.Log("ERROR: %v", err)
			continue
		}
		pending <- conn
	}
}

func (p *TCPProxy) Die() {
	p.die = true
	// fake request to trigger die
	if resp, err := http.Get("http://" + p.LocalAddrString); err == nil {
		resp.Body.Close()
	}
	<-p.dead
}

func (p *TCPProxy) Log(format string, args ...interface{}) {
	log.Printf("["+p.LocalAddrString+"][TCP]  "+format, args...)
}

func (p *TCPProxy) copy(dst io.ReadWriteCloser, src io.ReadWriteCloser) {
	io.Copy(dst, src)
	dst.Close()
	src.Close()
}

func (p *TCPProxy) handleConn(id int, in <-chan *net.TCPConn, die <-chan bool) {
	p.Log("initialized handler %d", id)
	for {
		select {
		case <-die:
			p.Log("handler die")
			return
		case lConn := <-in:
			rConn, err := net.DialTCP("tcp", nil, p.remoteAddr)
			if err != nil {
				p.Log("ERROR: %v", err)
				lConn.Close()
				continue
			}
			go p.copy(lConn, rConn)
			go p.copy(rConn, lConn)
		}
	}
}
