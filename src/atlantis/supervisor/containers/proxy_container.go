package containers

import (
	"atlantis/supervisor/docker"
	"atlantis/supervisor/rpc/types"
	"errors"
	"log"
)

const (
	ProxyFile = "proxy"
)

var (
	Proxy              *ProxyContainer
	ProxySSHPort       = uint16(22)
	ProxyConfigPort    = uint16(8080)
	ProxyMinExposePort = uint16(40000)
	ProxyMaxExposePort = uint16(65535)
	ProxyNumHandlers   = 16
	ProxyMaxPending    = 1024
	ProxyCPUShares     = int64(1)
	ProxyMemoryLimit   = int64(256)
)

type ProxyContainer struct {
	types.ProxyContainer
}

func UpdateProxy(host, sha string) error {
	if Proxy != nil && Proxy.Sha == sha {
		return errors.New("Already Deployed.")
	}
	newProxy := &ProxyContainer{ProxyContainer: types.ProxyContainer{
		Host:          host,
		ID:            "proxy-" + sha,
		App:           "proxy",
		Sha:           sha,
		SSHPort:       ProxySSHPort,
		ConfigPort:    ProxyConfigPort,
		MinExposePort: ProxyMinExposePort,
		MaxExposePort: ProxyMaxExposePort,
		NumHandlers:   ProxyNumHandlers,
		MaxPending:    ProxyMaxPending,
		CPUShares:     ProxyCPUShares,
		MemoryLimit:   ProxyMemoryLimit,
		VEthName:      "vethpxy"+sha[:6],
	}}
	if err := newProxy.Deploy(); err != nil { // deploy new proxy (this will update iptables as well)
		return err
	}
	if Proxy != nil { // if there is an older proxy, tear it down (this will update iptables as well)
		if err := Proxy.Teardown(); err != nil {
			log.Printf("ERROR: Teardown old proxy failed: %s", err.Error())
		}
	}
	Proxy = newProxy
	return nil
}

func (c *ProxyContainer) Deploy() error {
	err := docker.Deploy(c)
	if err != nil {
		return err
	}
	// TODO update iptables
	save() // save here because this is when we know the deployed container is actually alive
	return nil
}

// This calls the Teardown(id string) method to ensure that the ports/containers are freed. That will in turn
// call c.teardown(id string)
func (c *ProxyContainer) Teardown() error {
	// TODO update iptables
	return docker.Teardown(c)
}
