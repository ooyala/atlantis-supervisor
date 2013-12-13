package rpc

import (
	. "atlantis/common"
	"atlantis/supervisor/containers"
	. "atlantis/supervisor/rpc/types"
	"launchpad.net/gocheck"
	"os"
	"testing"
)

func Test(t *testing.T) { gocheck.TestingT(t) }

type RpcSuite struct{}

var _ = gocheck.Suite(&RpcSuite{})

func (s *RpcSuite) TestInit(c *gocheck.C) {
	c.Assert(Listen, gocheck.PanicMatches, "Not Initialized.")
	c.Assert(Init(":1337"), gocheck.IsNil)
}

func (s *RpcSuite) TestDeploy(c *gocheck.C) {
	os.Setenv("SUPERVISOR_PRETEND", "true")
	saveDir := "save_test"
	os.RemoveAll(saveDir)
	containers.Init("localhost", saveDir, 2, 2, 61000, 100, 1024)
	ih := new(Supervisor)
	arg := SupervisorDeployArg{}
	var reply SupervisorDeployReply
	c.Assert(ih.Deploy(arg, &reply), gocheck.ErrorMatches, "Please specify an app\\.")
	arg = SupervisorDeployArg{App: "theApp"}
	reply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(arg, &reply), gocheck.ErrorMatches, "Please specify a sha\\.")
	arg = SupervisorDeployArg{App: "theApp", Sha: "theSha"}
	reply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(arg, &reply), gocheck.ErrorMatches, "Please specify a container id\\.")
	arg = SupervisorDeployArg{App: "theApp", Sha: "theSha", ContainerID: "theContainerID"}
	reply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(arg, &reply), gocheck.ErrorMatches, "Please specify a manifest\\.")
	arg = SupervisorDeployArg{App: "theApp", Sha: "theSha", ContainerID: "theContainerID", Manifest: &Manifest{}}
	reply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(arg, &reply), gocheck.ErrorMatches, "Please specify a number of CPU shares\\.")
	arg = SupervisorDeployArg{App: "theApp", Sha: "theSha", ContainerID: "theContainerID", Manifest: &Manifest{CPUShares: 1}}
	reply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(arg, &reply), gocheck.ErrorMatches, "Please specify a memory limit\\.")
	arg = SupervisorDeployArg{App: "theApp", Sha: "theSha", ContainerID: "theContainerID", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	reply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Container.ID, gocheck.Equals, arg.ContainerID)
	c.Assert(reply.Container.PrimaryPort, gocheck.Equals, uint16(61000))
	c.Assert(reply.Container.SecondaryPorts, gocheck.DeepEquals, []uint16{61004, 61006})
	c.Assert(reply.Container.SSHPort, gocheck.Equals, uint16(61002))
	c.Assert(reply.Container.ID, gocheck.Equals, "theContainerID")
	c.Assert(reply.Container.App, gocheck.Equals, "theApp")
	c.Assert(reply.Container.Sha, gocheck.Equals, "theSha")
	c.Assert(reply.Container.DockerID, gocheck.Equals, "pretend-docker-id-61000")
	os.RemoveAll(saveDir)
}

func (s *RpcSuite) TestTeardown(c *gocheck.C) {
	os.Setenv("SUPERVISOR_PRETEND", "true")
	saveDir := "save_test"
	os.RemoveAll(saveDir)
	containers.Init("localhost", saveDir, 10, 2, 61000, 100, 1024)
	ih := new(Supervisor)
	// deploy a container
	var dreply SupervisorDeployReply
	darg := SupervisorDeployArg{App: "theApp", Sha: "theSha", ContainerID: "theContainerID", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	// teardown invalid args
	arg := SupervisorTeardownArg{}
	var reply SupervisorTeardownReply
	c.Assert(ih.Teardown(arg, &reply), gocheck.ErrorMatches, "Please specify container ids or all\\.")
	// teardown no container ids
	arg = SupervisorTeardownArg{ContainerIDs: []string{}}
	reply = SupervisorTeardownReply{}
	c.Assert(ih.Teardown(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Status, gocheck.Equals, StatusOk)
	c.Assert(reply.ContainerIDs, gocheck.DeepEquals, arg.ContainerIDs)
	// teardown nonexistant container id
	arg = SupervisorTeardownArg{ContainerIDs: []string{"doesntExist"}}
	reply = SupervisorTeardownReply{}
	c.Assert(ih.Teardown(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Status != StatusOk, gocheck.Equals, true) // gocheck doesn't have a NotEquals?!
	c.Assert(reply.ContainerIDs, gocheck.DeepEquals, []string{})
	// deploy more
	darg = SupervisorDeployArg{App: "theApp2", Sha: "theSha2", ContainerID: "theContainerID2", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	dreply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	darg = SupervisorDeployArg{App: "theApp3", Sha: "theSha3", ContainerID: "theContainerID3", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	dreply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	darg = SupervisorDeployArg{App: "theApp4", Sha: "theSha4", ContainerID: "theContainerID4", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	dreply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	darg = SupervisorDeployArg{App: "theApp5", Sha: "theSha5", ContainerID: "theContainerID5", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	dreply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	// teardown multiple container ids with one failure
	arg = SupervisorTeardownArg{ContainerIDs: []string{"doesntExist", "theContainerID", "theContainerID2"}}
	reply = SupervisorTeardownReply{}
	c.Assert(ih.Teardown(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Status != StatusOk, gocheck.Equals, true) // gocheck doesn't have a NotEquals?!
	c.Assert(reply.ContainerIDs, gocheck.DeepEquals, []string{"theContainerID", "theContainerID2"})
	// deploy more
	darg = SupervisorDeployArg{App: "theApp", Sha: "theSha", ContainerID: "theContainerID", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	dreply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	darg = SupervisorDeployArg{App: "theApp2", Sha: "theSha2", ContainerID: "theContainerID2", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	dreply = SupervisorDeployReply{}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	// teardown multiple container ids with no failure
	arg = SupervisorTeardownArg{ContainerIDs: []string{"theContainerID", "theContainerID2"}}
	reply = SupervisorTeardownReply{}
	c.Assert(ih.Teardown(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Status, gocheck.Equals, StatusOk) // gocheck doesn't have a NotEquals?!
	c.Assert(reply.ContainerIDs, gocheck.DeepEquals, []string{"theContainerID", "theContainerID2"})
	// teardown all
	arg = SupervisorTeardownArg{All: true}
	reply = SupervisorTeardownReply{}
	c.Assert(ih.Teardown(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Status, gocheck.Equals, StatusOk) // gocheck doesn't have a NotEquals?!
	c.Assert(reply.ContainerIDs, gocheck.DeepEquals, []string{"theContainerID3", "theContainerID4",
		"theContainerID5"})
	os.RemoveAll(saveDir)
}

func (s *RpcSuite) TestHealthCheck(c *gocheck.C) {
	os.Setenv("SUPERVISOR_PRETEND", "true")
	saveDir := "save_test"
	os.RemoveAll(saveDir)
	containers.Init("localhost", saveDir, 2, 2, 61000, 100, 1024)
	ih := new(Supervisor)
	// check health
	arg := SupervisorHealthCheckArg{}
	var reply SupervisorHealthCheckReply
	c.Assert(ih.HealthCheck(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Status, gocheck.Equals, StatusOk)
	c.Assert(reply.Containers.Total, gocheck.Equals, uint(2))
	c.Assert(reply.Containers.Free, gocheck.Equals, uint(2))
	c.Assert(reply.Containers.Used, gocheck.Equals, uint(0))
	// deploy one
	var dreply SupervisorDeployReply
	darg := SupervisorDeployArg{App: "theApp1", Sha: "theSha1", ContainerID: "theContainerID1", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	// check health
	arg = SupervisorHealthCheckArg{}
	reply = SupervisorHealthCheckReply{}
	c.Assert(ih.HealthCheck(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Status, gocheck.Equals, StatusOk)
	c.Assert(reply.Containers.Total, gocheck.Equals, uint(2))
	c.Assert(reply.Containers.Free, gocheck.Equals, uint(1))
	c.Assert(reply.Containers.Used, gocheck.Equals, uint(1))
	// deploy another
	dreply = SupervisorDeployReply{}
	darg = SupervisorDeployArg{App: "theApp2", Sha: "theSha2", ContainerID: "theContainerID2", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	// check health
	arg = SupervisorHealthCheckArg{}
	reply = SupervisorHealthCheckReply{}
	c.Assert(ih.HealthCheck(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Status, gocheck.Equals, StatusFull)
	c.Assert(reply.Containers.Total, gocheck.Equals, uint(2))
	c.Assert(reply.Containers.Free, gocheck.Equals, uint(0))
	c.Assert(reply.Containers.Used, gocheck.Equals, uint(2))
	// teardown all
	targ := SupervisorTeardownArg{All: true}
	var treply SupervisorTeardownReply
	c.Assert(ih.Teardown(targ, &treply), gocheck.IsNil)
	// check health
	arg = SupervisorHealthCheckArg{}
	reply = SupervisorHealthCheckReply{}
	c.Assert(ih.HealthCheck(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Status, gocheck.Equals, StatusOk)
	c.Assert(reply.Containers.Total, gocheck.Equals, uint(2))
	c.Assert(reply.Containers.Free, gocheck.Equals, uint(2))
	c.Assert(reply.Containers.Used, gocheck.Equals, uint(0))
	os.RemoveAll(saveDir)
}

func (s *RpcSuite) TestList(c *gocheck.C) {
	os.Setenv("SUPERVISOR_PRETEND", "true")
	saveDir := "save_test"
	os.RemoveAll(saveDir)
	containers.Init("localhost", saveDir, 2, 2, 61000, 100, 1024)
	ih := new(Supervisor)
	// list
	arg := SupervisorListArg{}
	var reply SupervisorListReply
	c.Assert(ih.List(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Containers, gocheck.DeepEquals, map[string]*Container{})
	c.Assert(reply.UnusedPorts, gocheck.DeepEquals, []uint16{61000, 61001})
	// deploy one
	var dreply SupervisorDeployReply
	darg := SupervisorDeployArg{App: "theApp1", Sha: "theSha1", ContainerID: "theContainerID1", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	container1 := *dreply.Container
	// list
	arg = SupervisorListArg{}
	reply = SupervisorListReply{}
	c.Assert(ih.List(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Containers, gocheck.DeepEquals, map[string]*Container{container1.ID: &container1})
	c.Assert(reply.UnusedPorts, gocheck.DeepEquals, []uint16{61001})
	// deploy another
	dreply = SupervisorDeployReply{}
	darg = SupervisorDeployArg{App: "theApp2", Sha: "theSha2", ContainerID: "theContainerID2", Manifest: &Manifest{CPUShares: 1, MemoryLimit: 1}}
	c.Assert(ih.Deploy(darg, &dreply), gocheck.IsNil)
	container2 := *dreply.Container
	// list
	arg = SupervisorListArg{}
	reply = SupervisorListReply{}
	c.Assert(ih.List(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Containers, gocheck.DeepEquals, map[string]*Container{container1.ID: &container1,
		container2.ID: &container2})
	c.Assert(reply.UnusedPorts, gocheck.DeepEquals, []uint16{})
	// teardown all
	targ := SupervisorTeardownArg{All: true}
	var treply SupervisorTeardownReply
	c.Assert(ih.Teardown(targ, &treply), gocheck.IsNil)
	// list
	arg = SupervisorListArg{}
	reply = SupervisorListReply{}
	c.Assert(ih.List(arg, &reply), gocheck.IsNil)
	c.Assert(reply.Containers, gocheck.DeepEquals, map[string]*Container{})
	c.Assert(reply.UnusedPorts, gocheck.DeepEquals, []uint16{61000, 61001})
	os.RemoveAll(saveDir)
}
