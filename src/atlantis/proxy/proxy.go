package proxy

import (
	. "atlantis/proxy/types"
)

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
