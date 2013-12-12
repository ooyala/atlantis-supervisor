package proxy

import (
	"io"
	"log"
	"net"
)

type ProxyConfig struct {
	NumHandlers      int
	MaxPending       int
	LocalAddr        string
	RemoteAddr       string
}

type Proxy struct {
	NumHandlers      int
	MaxPending       int
	LocalAddrString  string       `json:"LocalAddr"`
	LocalAddr        *net.TCPAddr `json:"-"`
	RemoteAddrString string       `json:"RemoteAddr"`
	RemoteAddr       *net.TCPAddr `json:"-"`
	dead             chan bool
	die              bool
}

func NewProxyWithConfig(c *ProxyConfig) (*Proxy, error) {
	return NewProxy(c.LocalAddr, c.RemoteAddr, c.NumHandlers, c.MaxPending)
}

func NewProxy(lAddr, rAddr string, numHandlers, maxPending int) (*Proxy, error) {
	localAddr, err := net.ResolveTCPAddr("tcp", lAddr)
	if err != nil {
		return nil, err
	}
	remoteAddr, err := net.ResolveTCPAddr("tcp", rAddr)
	if err != nil {
		return nil, err
	}
	return &Proxy{
		NumHandlers:      numHandlers,
		MaxPending:       maxPending,
		LocalAddrString:  lAddr,
		LocalAddr:        localAddr,
		RemoteAddrString: rAddr,
		RemoteAddr:       remoteAddr,
	}, nil
}

func (p *Proxy) Listen() {
	p.Log("Proxying to %s", p.RemoteAddrString)
	p.dead = make(chan bool)

	listener, err := net.ListenTCP("tcp", p.LocalAddr)
	if err != nil {
		panic(err)
	}

	pending := make(chan *net.TCPConn, p.MaxPending)
	die := make(chan bool, p.NumHandlers)

	for i := 0; i < p.NumHandlers; i++ {
		go p.handleConn(i, pending, die)
	}

	for {
		if p.die {
			p.Log("Die")
			if err := listener.Close(); err != nil {
				p.Log("ERROR: " + err.Error())
			}
			for i := 0; i < p.NumHandlers; i++ {
				die <- true
			}
			p.dead <- true
			return
		}
		conn, err := listener.AcceptTCP()
		if err != nil {
			p.Log("ERROR: %v", err)
			continue
		}
		pending <- conn
	}
}

func (p *Proxy) Log(format string, args ...interface{}) {
	log.Printf("["+p.LocalAddrString+"] "+format, args...)
}

func (p *Proxy) copy(id int, dst io.ReadWriteCloser, src io.ReadWriteCloser) {
	io.Copy(dst, src)
	dst.Close()
	src.Close()
}

func (p *Proxy) handleConn(id int, in <-chan *net.TCPConn, die <-chan bool) {
	p.Log("Initialized Handler %d", id)
	for {
		select {
		case <-die:
			p.Log("Handler Die")
			return
		case lConn := <-in:
			rConn, err := net.DialTCP("tcp", nil, p.RemoteAddr)
			if err != nil {
				p.Log("ERROR: %v", err)
				lConn.Close()
				continue
			}
			go p.copy(0, lConn, rConn)
			go p.copy(1, rConn, lConn)
		}
	}
}
