package containers

import (
	"atlantis/supervisor/docker"
	"atlantis/supervisor/rpc/types"
)

type Container struct {
	types.Container
}

// Deploy the given app+sha with the dependencies defined in deps. This will spin up a new docker container.
func (c *Container) Deploy(host, app, sha, env string) error {
	c.Host = host
	c.App = app
	c.Sha = sha
	c.Env = env
	err := docker.Deploy(c)
	if err != nil {
		return err
	}
	save() // save here because this is when we know the deployed container is actually alive
	return nil
}

// This calls the Teardown(id string) method to ensure that the ports/containers are freed. That will in turn
// call c.teardown(id string)
func (c *Container) Teardown() {
	Teardown(c.ID)
}
