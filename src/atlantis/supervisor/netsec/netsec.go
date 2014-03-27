package netsec

import (
	"errors"
	"sync"
)

type ContainerSecurity struct {
	veth           string
	mark           int
	ID             string
	Pid            int
	SecurityGroups map[string]uint16 // ipgroup name -> port
}

func (c *ContainerSecurity) GetNetworkInfo() error {
	// this will call out to guano to figure out what the veth and mark should be
	// TODO[jigish] implement
	return errors.New("unimplemented")
}

type NetworkSecurity struct {
	sync.Mutex
	deniedIPs  map[string]bool              // list of denied IPs. map for easy existence check
	ipGroups   map[string][]string          // group name -> list of infrastructure IPs to blanket deny
	containers map[string]*ContainerSecurty // container id -> ContainerSecurity
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
		if deniedIPs[ip] {
			// already exists
			continue
		}
		if err := n.blanketDeny(true, ip); err != nil {
			return err
		}
		deniedIPs[ip] = true
		newIPs = append(newIPs, ip)
	}
	// remove blanket deny rule for removed IPs
	for _, ip := range toRemove {
		if !deniedIPs[ip] {
			// already removed
			continue
		}
		n.blanketDeny(false, ip)
		delete(deniedIPs, ip)
	}
	// add/remove forward rules for new IPs for everything that uses the name
	for _, contSec := range n.containers {
		port, exists := contSec.SecurityGroups[name]
		if !exists {
			// this container does not use this ip group
			continue
		}
		// add new ips
		for _, ip := range newIPs {
			n.openPort(true, contSec.mark, ip, port)
		}
		// remove old ips
		for _, ip := range toRemove {
			n.openPort(false, contSec.mark, ip, port)
		}
	}
	// update ipGroups
	n.ipGroups[name] = ips
	return nil
}

func (n *NetworkSecurity) DeleteIPGroup(name string) error {
	if err := n.UpdateIPGroup(name, []string{}); err != nil {
		return err
	}
	n.Lock()
	defer n.Unlock()
	delete(n.ipGroups, name)
	return nil
}

func (n *NetworkSecurity) AddContainerSecurity(id string, pid int, sgs map[string]uint16) error {
	n.Lock()
	defer n.Unlock()
	if _, exists := n.containers[id]; exists {
		// we already have security set up for this id. don't do it and return an error.
		return errors.New("Container "+id+" already has Network Security set up.")
	}
	// make sure all groups exist
	for group, _ := range sgs {
		_, exists := n.ipGroups[group]
		if !exists {
			return errors.New("IP Group "+group+" does not exist")
		}
	}

	// fetch network info
	contSec := &ContainerSecurity{
		ID: id,
		Pid: pid,
		SecurityGroups: sgs
	}
	if err := contSec.GetNetworkInfo(); err != nil {
		return err
	}
	// create prerouting mark
	if err := n.markVeth(true, contSec.veth, contSec.mark); err != nil {
		return err
	}
	// add forward rules
	for group, port := range sgs {
		ips := n.ipGroups[group]
		for _, ip := range ips {
			if err := n.openPort(true, contSec.mark, ip, port); err != nil {
				defer n.RemoveContainerSecurity(id) // cleanup created references when we error out
				return err
			}
		}
	}
}

func (n *NetworkSecurity) RemoveContainerSecurity(id string) error {
	n.Lock()
	defer n.Unlock()
	var (
		contSec *ContainerSecurity
		exists bool
	)
	if contSec, exists = n.containers[id]; !exists {
		// no container security here, nothing to remove
		return nil
	}
	// remove prerouting mark
	n.markVeth(false, contSec.veth, contSec.mark)
	// remove forward rules
	for group, port := range sgs {
		ips := n.ipGroups[group]
		for _, ip := range ips {
			n.openPort(false, contSec.mark, ip, port)
		}
	}
	return nil
}

func (n *NetworkSecurity) blanketDeny(add bool, ip string) error {
	// TODO[jigish] implement
	return errors.New("unimplemented")
}

func (n *NetworkSecurity) openPort(add bool, mark int, ip string, port uint16) error {
	// TODO[jigish] implement
	return errors.New("unimplemented")
}

func (n *NetworkSecurity) markVeth(add bool, veth string, mark int) error {
	// TODO[jigish] implement
	return errors.New("unimplemented")
}
