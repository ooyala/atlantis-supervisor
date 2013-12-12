package proxy

const (
	ProxyTypeTCP = iota
	ProxyTypeHTTP
)

type ProxyConfig struct {
	Type        int
	NumHandlers int
	MaxPending  int
	LocalAddr   string
	RemoteAddr  string
}

type Proxy interface {
	Init() error
	Listen()
	Die()
	LocalAddr() string
	RemoteAddr() string
}

func NewProxyWithConfig(c *ProxyConfig) Proxy {
	switch c.Type {
	case ProxyTypeHTTP:
		return NewHTTPProxy(c.LocalAddr, c.RemoteAddr)
	case ProxyTypeTCP:
		return NewTCPProxy(c.LocalAddr, c.RemoteAddr, c.NumHandlers, c.MaxPending)
	default:
		return nil
	}
}
