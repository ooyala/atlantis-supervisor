package containers

import (
	"atlantis/supervisor/rpc/types"
	"launchpad.net/gocheck"
	"os"
	"testing"
)

func TestContainers(t *testing.T) { gocheck.TestingT(t) }

type ContainersSuite struct{}

var _ = gocheck.Suite(&ContainersSuite{})

func (s *ContainersSuite) TestInit(c *gocheck.C) {
	// Positive test
	saveDir := "save_test"
	os.RemoveAll(saveDir)
	c.Assert(Init("localhost", saveDir, uint16(100), uint16(5), uint16(61000), 100, 1024), gocheck.IsNil)
	if _, err := os.Stat(saveDir); err != nil {
		c.Fatal("Init did not make the save directory")
	}
	os.RemoveAll(saveDir)
	dieChan <- true

	// Negative test for invalid config
	c.Assert(Init("localhost", saveDir, uint16(1000), uint16(5), uint16(61000), 100, 1024), gocheck.ErrorMatches, "Invalid Config.+")
	if _, err := os.Stat(saveDir); err == nil {
		os.RemoveAll(saveDir)
		c.Fatal("Init should not make the save directory if config is bad.")
	}

	// Negative test for invalid config
	c.Assert(Init("localhost", saveDir, uint16(65535), uint16(65535), uint16(65535), 100, 1024), gocheck.ErrorMatches, "Invalid Config.+")
	if _, err := os.Stat(saveDir); err == nil {
		os.RemoveAll(saveDir)
		c.Fatal("Init should not make the save directory if config is bad.")
	}
	os.RemoveAll(saveDir)
}

func (s *ContainersSuite) TestReserve(c *gocheck.C) {
	saveDir := "save_test"
	os.RemoveAll(saveDir)
	c.Assert(Init("localhost", saveDir, uint16(2), uint16(2), uint16(61000), 100, 1024), gocheck.IsNil)
	// First reserve should work
	container, err := Reserve("first", &types.Manifest{CPUShares: 50, MemoryLimit: 512})
	c.Assert(err, gocheck.IsNil)
	c.Assert(container.PrimaryPort, gocheck.Equals, uint16(61000))
	c.Assert(container.SecondaryPorts, gocheck.DeepEquals, []uint16{61004, 61006})
	c.Assert(container.SSHPort, gocheck.Equals, uint16(61002))
	c.Assert(container.ID, gocheck.Equals, "first")
	c.Assert(container.App, gocheck.Equals, "")
	c.Assert(container.Sha, gocheck.Equals, "")
	c.Assert(container.DockerID, gocheck.Equals, "")
	// Second should fail because id is taken
	container, err = Reserve("first", &types.Manifest{CPUShares: 1, MemoryLimit: 1})
	c.Assert(err, gocheck.ErrorMatches, "The ID \\(first\\) is in use\\.")
	// Third should fail because not enough CPU shares
	container, err = Reserve("third", &types.Manifest{CPUShares: 51, MemoryLimit: 1})
	c.Assert(err, gocheck.ErrorMatches, "Not enough CPU Shares to reserve. \\(51 requested, 50 available\\)")
	// Fourth should fail because not enough CPU shares
	container, err = Reserve("fourth", &types.Manifest{CPUShares: 1, MemoryLimit: 513})
	c.Assert(err, gocheck.ErrorMatches, "Not enough Memory to reserve. \\(513 requested, 512 available\\)")
	// Fifth should work
	container, err = Reserve("fifth", &types.Manifest{CPUShares: 1, MemoryLimit: 1})
	c.Assert(err, gocheck.IsNil)
	c.Assert(container.PrimaryPort, gocheck.Equals, uint16(61001))
	c.Assert(container.SecondaryPorts, gocheck.DeepEquals, []uint16{61005, 61007})
	c.Assert(container.SSHPort, gocheck.Equals, uint16(61003))
	c.Assert(container.ID, gocheck.Equals, "fifth")
	c.Assert(container.App, gocheck.Equals, "")
	c.Assert(container.Sha, gocheck.Equals, "")
	c.Assert(container.DockerID, gocheck.Equals, "")
	// Sixth should fail because there are no containers
	container, err = Reserve("sixth", &types.Manifest{CPUShares: 1, MemoryLimit: 1})
	c.Assert(err, gocheck.ErrorMatches, "No free containers to reserve\\.")
	os.RemoveAll(saveDir)
	dieChan <- true
}

func (s *ContainersSuite) TestTeardown(c *gocheck.C) {
	os.Setenv("SUPERVISOR_PRETEND", "true")
	saveDir := "save_test"
	os.RemoveAll(saveDir)
	c.Assert(Init("localhost", saveDir, uint16(2), uint16(2), uint16(61000), 100, 1024), gocheck.IsNil)
	// First reserve should work
	_, err := Reserve("first", &types.Manifest{CPUShares: 1, MemoryLimit: 1})
	c.Assert(err, gocheck.IsNil)
	// Second should fail because id is taken
	_, err = Reserve("first", &types.Manifest{CPUShares: 1, MemoryLimit: 1})
	c.Assert(err, gocheck.ErrorMatches, "The ID \\(first\\) is in use\\.")
	// Teardown
	c.Assert(Teardown("first"), gocheck.Equals, true)
	// Third should work
	_, err = Reserve("first", &types.Manifest{CPUShares: 1, MemoryLimit: 1})
	c.Assert(err, gocheck.IsNil)
	os.RemoveAll(saveDir)
	dieChan <- true
}

func (s *ContainersSuite) TestList(c *gocheck.C) {
	os.Setenv("SUPERVISOR_PRETEND", "true")
	saveDir := "save_test"
	os.RemoveAll(saveDir)
	c.Assert(Init("localhost", saveDir, uint16(2), uint16(2), uint16(61000), 100, 1024), gocheck.IsNil)
	// reserve first and list
	first, err := Reserve("first", &types.Manifest{CPUShares: 1, MemoryLimit: 1})
	c.Assert(err, gocheck.IsNil)
	conts, ports := List()
	c.Assert(*conts["first"], gocheck.DeepEquals, first.Container)
	c.Assert(ports, gocheck.DeepEquals, []uint16{61001})
	// reserve second and list
	second, err := Reserve("second", &types.Manifest{CPUShares: 2, MemoryLimit: 2})
	c.Assert(err, gocheck.IsNil)
	conts, ports = List()
	c.Assert(*conts["second"], gocheck.DeepEquals, second.Container)
	c.Assert(ports, gocheck.DeepEquals, []uint16{})
	// teardown first and list
	c.Assert(Teardown("first"), gocheck.Equals, true)
	conts, ports = List()
	_, present := conts["first"]
	c.Assert(present, gocheck.Equals, false)
	c.Assert(ports, gocheck.DeepEquals, []uint16{61000})
	os.RemoveAll(saveDir)
	dieChan <- true
}

func (s *ContainersSuite) TestNums(c *gocheck.C) {
	os.Setenv("SUPERVISOR_PRETEND", "true")
	saveDir := "save_test"
	os.RemoveAll(saveDir)
	c.Assert(Init("localhost", saveDir, uint16(2), uint16(2), uint16(61000), 100, 1024), gocheck.IsNil)
	// reserve first and list
	_, err := Reserve("first", &types.Manifest{CPUShares: 1, MemoryLimit: 100})
	c.Assert(err, gocheck.IsNil)
	cont, cpu, mem := Nums()
	c.Assert(cont.Total, gocheck.Equals, uint(2))
	c.Assert(cont.Used, gocheck.Equals, uint(1))
	c.Assert(cont.Free, gocheck.Equals, uint(1))
	c.Assert(cpu.Total, gocheck.Equals, uint(100))
	c.Assert(cpu.Used, gocheck.Equals, uint(1))
	c.Assert(cpu.Free, gocheck.Equals, uint(99))
	c.Assert(mem.Total, gocheck.Equals, uint(1024))
	c.Assert(mem.Used, gocheck.Equals, uint(100))
	c.Assert(mem.Free, gocheck.Equals, uint(924))
	// reserve second and list
	_, err = Reserve("second", &types.Manifest{CPUShares: 2, MemoryLimit: 200})
	c.Assert(err, gocheck.IsNil)
	cont, cpu, mem = Nums()
	c.Assert(cont.Total, gocheck.Equals, uint(2))
	c.Assert(cont.Used, gocheck.Equals, uint(2))
	c.Assert(cont.Free, gocheck.Equals, uint(0))
	c.Assert(cpu.Total, gocheck.Equals, uint(100))
	c.Assert(cpu.Used, gocheck.Equals, uint(3))
	c.Assert(cpu.Free, gocheck.Equals, uint(97))
	c.Assert(mem.Total, gocheck.Equals, uint(1024))
	c.Assert(mem.Used, gocheck.Equals, uint(300))
	c.Assert(mem.Free, gocheck.Equals, uint(724))
	// teardown first and list
	c.Assert(Teardown("first"), gocheck.Equals, true)
	cont, cpu, mem = Nums()
	c.Assert(cont.Total, gocheck.Equals, uint(2))
	c.Assert(cont.Used, gocheck.Equals, uint(1))
	c.Assert(cont.Free, gocheck.Equals, uint(1))
	c.Assert(cpu.Total, gocheck.Equals, uint(100))
	c.Assert(cpu.Used, gocheck.Equals, uint(2))
	c.Assert(cpu.Free, gocheck.Equals, uint(98))
	c.Assert(mem.Total, gocheck.Equals, uint(1024))
	c.Assert(mem.Used, gocheck.Equals, uint(200))
	c.Assert(mem.Free, gocheck.Equals, uint(824))
	os.RemoveAll(saveDir)
	dieChan <- true
}
