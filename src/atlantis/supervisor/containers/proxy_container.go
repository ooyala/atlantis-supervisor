package containers

import (
	ptypes "atlantis/proxy/types"
	"atlantis/supervisor/docker"
	"atlantis/supervisor/rpc/types"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

func ConfigureProxy(cfg map[string]*ptypes.ProxyConfig) error {
	if Proxy == nil {
		return errors.New("No Proxy Deployed.")
	}
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	cfgBuffer := bytes.NewBuffer(cfgBytes)
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s:%d", Proxy.IP, Proxy.ConfigPort), cfgBuffer)
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	} else {
		return errors.New(string(bodyBytes))
	}
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
		VEthName:      "vethpxy" + sha[:6],
	}}
	if err := newProxy.Deploy(); err != nil { // deploy new proxy (this will update iptables as well)
		return err
	}
	if Proxy != nil { // if there is an older proxy, tear it down (this will update iptables as well)
		if err := Proxy.Teardown(); err != nil {
			// TODO[jigish] email?
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
