package types

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
