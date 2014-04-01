package netsec

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func guano(pid int) (string, string, error) {
	binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", "", err
	}
	out, err := executeCommand(path.Join(binDir, "guano"), fmt.Sprintf("%d", pid), "eth0")
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

func (c ContainerSecurity) String() string {
	return fmt.Sprintf("veth %s mark %s id %s pid %d groups %v", c.veth, c.mark, c.ID, c.Pid, c.SecurityGroups)
}

func NewContainerSecurity(id string, pid int, sgs map[string][]uint16) (contSec *ContainerSecurity, err error) {
	contSec = &ContainerSecurity{
		ID:             id,
		Pid:            pid,
		SecurityGroups: sgs,
	}
	for i := 0; i < 5 && err != nil; i++ {
		contSec.mark, contSec.veth, err = guano(pid)
		time.Sleep(1 * time.Second)
	}
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

func (c *ContainerSecurity) allowPort(ip string, port uint16) error {
	return c.filterPort("-I", ip, port)
}

func (c *ContainerSecurity) rejectPort(ip string, port uint16) error {
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
