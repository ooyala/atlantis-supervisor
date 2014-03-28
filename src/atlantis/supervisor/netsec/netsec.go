package netsec

import (
	"atlantis/supervisor/containers/serialize"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

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

func guano(pid int) (string, string, error) {
	out, err := ExecuteCommand("/usr/sbin/guano", fmt.Sprintf("%d", pid), "eth0")
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(out, " ")
	if len(parts) != 2 {
		return "", "", errors.New("guano is batshit insane.")
	}
	return parts[0], parts[1], nil
}

type NetworkSecurity struct {
	sync.Mutex
	SaveFile   string                        // where to save state
	deniedIPs  map[string]bool               // list of denied IPs. map for easy existence check
	ipGroups   map[string][]string           // group name -> list of infrastructure IPs to blanket deny
	containers map[string]*ContainerSecurity // container id -> ContainerSecurity
}

func New(saveFile string) *NetworkSecurity {
	return &NetworkSecurity{
		Mutex:      sync.Mutex{},
		SaveFile:   saveFile,
		deniedIPs:  map[string]bool{},
		ipGroups:   map[string][]string{},
		containers: map[string]*ContainerSecurity{},
	}
}

func (n *NetworkSecurity) UpdateIPGroup(name string, ips []string) error {
	n.Lock()
	defer n.Unlock()
	current, exists := n.ipGroups[name]
	toRemove := []string{}
	if exists {
		// figure out what is being removed
		incomingMap := map[string]bool{}
		for _, ip := range ips {
			incomingMap[ip] = true
		}
		for _, ip := range current {
			if !incomingMap[ip] {
				toRemove = append(toRemove, ip)
			}
		}
	}
	// add blanket deny rule for new IPs
	newIPs := []string{}
	for _, ip := range ips {
		if n.deniedIPs[ip] {
			// already exists
			continue
		}
		if err := n.rejectIP(true, ip); err != nil {
			return err
		}
		n.deniedIPs[ip] = true
		newIPs = append(newIPs, ip)
	}
	// remove blanket deny rule for removed IPs
	for _, ip := range toRemove {
		if !n.deniedIPs[ip] {
			// already removed
			continue
		}
		n.rejectIP(false, ip)
		delete(n.deniedIPs, ip)
	}
	// add/remove forward rules for new IPs for everything that uses the name
	for _, contSec := range n.containers {
		ports, exists := contSec.SecurityGroups[name]
		if !exists {
			// this container does not use this ip group
			continue
		}
		for _, port := range ports {
			// add new ips
			for _, ip := range newIPs {
				n.openPort(true, contSec.mark, ip, port)
			}
			// remove old ips
			for _, ip := range toRemove {
				n.openPort(false, contSec.mark, ip, port)
			}
		}
	}
	// update ipGroups
	n.ipGroups[name] = ips
	n.save()
	return nil
}

func (n *NetworkSecurity) DeleteIPGroup(name string) error {
	if err := n.UpdateIPGroup(name, []string{}); err != nil {
		return err
	}
	n.Lock()
	defer n.Unlock()
	delete(n.ipGroups, name)
	n.save()
	return nil
}

func (n *NetworkSecurity) AddContainerSecurity(id string, pid int, sgs map[string][]uint16) error {
	n.Lock()
	defer n.Unlock()
	if _, exists := n.containers[id]; exists {
		// we already have security set up for this id. don't do it and return an error.
		return errors.New("Container " + id + " already has Network Security set up.")
	}
	// make sure all groups exist
	for group, _ := range sgs {
		_, exists := n.ipGroups[group]
		if !exists {
			return errors.New("IP Group " + group + " does not exist")
		}
	}

	// fetch network info
	contSec, err := NewContainerSecurity(id, pid, sgs)
	if err != nil {
		return err
	}
	// create prerouting mark
	if err := n.markVeth(true, contSec.veth, contSec.mark); err != nil {
		return err
	}
	// add forward rules
	for group, ports := range sgs {
		for _, port := range ports {
			ips := n.ipGroups[group]
			for _, ip := range ips {
				if err := n.openPort(true, contSec.mark, ip, port); err != nil {
					defer n.RemoveContainerSecurity(id) // cleanup created references when we error out
					return err
				}
			}
		}
	}
	n.save()
	return nil
}

func (n *NetworkSecurity) RemoveContainerSecurity(id string) error {
	n.Lock()
	defer n.Unlock()
	var (
		contSec *ContainerSecurity
		exists  bool
	)
	if contSec, exists = n.containers[id]; !exists {
		// no container security here, nothing to remove
		return nil
	}
	// remove prerouting mark
	n.markVeth(false, contSec.veth, contSec.mark)
	// remove forward rules
	for group, ports := range contSec.SecurityGroups {
		for _, port := range ports {
			ips := n.ipGroups[group]
			for _, ip := range ips {
				n.openPort(false, contSec.mark, ip, port)
			}
		}
	}
	n.save()
	return nil
}

func (n *NetworkSecurity) rejectIP(add bool, ip string) error {
	action := "-I"
	if !add {
		action = "-D"
	}
	_, err := ExecuteCommand("iptables", action, "FORWARD", "-d", ip, "-j", "REJECT")
	return err
}

func (n *NetworkSecurity) openPort(add bool, mark string, ip string, port uint16) error {
	action := "-I"
	if !add {
		action = "-D"
	}
	_, err := ExecuteCommand("iptables", action, "FORWARD",
		"-d", ip,
		"-p", "tcp", "--dport", fmt.Sprintf("%d", port),
		"-m", "mark", "--mark", mark,
		"-j", "ACCEPT")
	return err
}

func (n *NetworkSecurity) markVeth(add bool, veth string, mark string) error {
	action := "-I"
	if !add {
		action = "-D"
	}
	_, err := ExecuteCommand("iptables", action, "PREROUTING", "-t", "mangle",
		"-m", "physdev", "--physdev-in", veth,
		"-j", "MARK", "--set-mark", mark)
	return err
}

func ExecuteCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		switch err.(type) {
		case *exec.ExitError:
			return string(out), errors.New(string(out))
		default:
			return string(out), err
		}
	}
	return string(out), nil
}

func (n *NetworkSecurity) save() {
	// save state
	serialize.SaveAll(serialize.SaveDefinition{
		n.SaveFile,
		n,
	})
}
