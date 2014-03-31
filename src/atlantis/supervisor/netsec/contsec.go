package netsec

import (
	"errors"
	"fmt"
	"strings"
)

func guano(pid int) (string, string, error) {
	out, err := executeCommand("/usr/sbin/guano", fmt.Sprintf("%d", pid), "eth0")
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(out, " ")
	if len(parts) != 2 {
		return "", "", errors.New("guano is batshit insane.")
	}
	return parts[0], parts[1], nil
}

type ContainerSecurity struct {
	veth           string
	mark           string
	ID             string
	Pid            int
	SecurityGroups map[string][]uint16 // ipgroup name -> ports
}

func NewContainerSecurity(id string, pid int, sgs map[string][]uint16) (contSec *ContainerSecurity, err error) {
	contSec = &ContainerSecurity{
		ID:             id,
		Pid:            pid,
		SecurityGroups: sgs,
	}
	contSec.mark, contSec.veth, err = guano(pid)
	return contSec, err
}

func (c *ContainerSecurity) filterPort(action, ip string, port uint16) error {
	_, err := executeCommand("iptables", action, "FORWARD",
		"-d", ip,
		"-p", "tcp", "--dport", fmt.Sprintf("%d", port),
		"-m", "mark", "--mark", c.mark,
		"-j", "ACCEPT")
	return err
}

func (c *ContainerSecurity) openPort(ip string, port uint16) error {
	return c.filterPort("-I", ip, port)
}

func (c *ContainerSecurity) closePort(ip string, port uint16) error {
	return c.filterPort("-D", ip, port)
}

func (c *ContainerSecurity) markVeth(action string) error {
	_, err := executeCommand("iptables", action, "PREROUTING", "-t", "mangle",
		"-m", "physdev", "--physdev-in", c.veth,
		"-j", "MARK", "--set-mark", c.mark)
	return err
}

func (c *ContainerSecurity) addMark() error {
	return c.markVeth("-I")
}

func (c *ContainerSecurity) delMark() error {
	return c.markVeth("-D")
}

