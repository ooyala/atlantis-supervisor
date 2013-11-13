package containers

import (
	"atlantis/supervisor/rpc/types"
	"launchpad.net/gocheck"
	"os"
	"testing"
)

func TestDocker(t *testing.T) { gocheck.TestingT(t) }

type DockerSuite struct{}

var _ = gocheck.Suite(&DockerSuite{})

func (s *DockerSuite) TestDeploy(c *gocheck.C) {
	os.Setenv("SUPERVISOR_PRETEND", "true")
	saveDir := "save_test"
	os.RemoveAll(saveDir)
	cont := &Container{Manifest: &types.Manifest{}}
	cont.Id = "first"
	cont.Deploy("theHost", "theApp", "theSha", "theEnv")
	c.Assert(cont.Host, gocheck.Equals, "theHost")
	c.Assert(cont.App, gocheck.Equals, "theApp")
	c.Assert(cont.Sha, gocheck.Equals, "theSha")
	c.Assert(cont.Env, gocheck.Equals, "theEnv")
	os.RemoveAll(saveDir)
}

// no testing for teardown because pretend teardown does nothing.
