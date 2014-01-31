/* Copyright 2014 Ooyala, Inc. All rights reserved.
 *
 * This file is licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package docker

import (
	"atlantis/supervisor/helper"
	"atlantis/supervisor/rpc/types"
	atypes "atlantis/types"
	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
)

var (
	RegistryHost   string
	dockerIDRegexp = regexp.MustCompile("^[A-Za-z0-9]+$")
	dockerLock     = sync.Mutex{}
	dockerClient   *docker.Client
)

func Init(registry string) (err error) {
	RegistryHost = registry
	dockerClient, err = docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		return err
	}
	go removeExited()
	go restartGhost()
	return nil
}

func pretending() bool {
	return os.Getenv("SUPERVISOR_PRETEND") != ""
}

func removeExited() {
	if pretending() {
		return
	}
	dockerLock.Lock()
	defer dockerLock.Unlock()
	containers, err := dockerClient.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		log.Printf("[RemoveExited] could not list containers: %v", err)
		return
	}
	for _, cont := range containers {
		log.Printf("[RemoveExited] checking %s (%v) : %s", cont.ID, cont.Names, cont.Status)
		if !strings.HasPrefix(cont.Status, "Exit") {
			continue
		}
		log.Printf("[RemoveExited] remove %s (%v)", cont.ID, cont.Names)
		err := dockerClient.RemoveContainer(cont.ID)
		if err != nil {
			log.Printf("[RemoveExited] -> error: %v", err)
		} else {
			log.Printf("[RemoveExited] -> success")
		}
	}
}

func restartGhost() {
	if pretending() {
		return
	}
	dockerLock.Lock()
	defer dockerLock.Unlock()
	containers, err := dockerClient.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		log.Printf("[RestartGhost] could not list containers: %v", err)
		return
	}
	for _, cont := range containers {
		log.Printf("[RestartGhost] checking %s (%v) : %s", cont.ID, cont.Names, cont.Status)
		if !strings.HasPrefix(cont.Status, "Ghost") {
			continue
		}
		log.Printf("[RestartGhost] restart %s (%v)", cont.ID, cont.Names)
		err := dockerClient.RestartContainer(cont.ID, 0)
		if err != nil {
			log.Printf("[RestartGhost] -> error: %v", err)
		} else {
			log.Printf("[RestartGhost] -> success")
		}
	}
}

func DockerCfgs(c types.GenericContainer) (*docker.Config, *docker.HostConfig) {
	switch typedC := c.(type) {
	case *types.Container:
		return ContainerDockerCfgs(typedC)
	default:
		return nil, nil
	}
}

func AppCfgs(c types.GenericContainer) (*atypes.AppConfig, error) {
	switch typedC := c.(type) {
	case *types.Container:
		return ContainerAppCfgs(typedC)
	default:
		return nil, errors.New("could not fetch app configs")
	}
}

func Deploy(c types.GenericContainer) error {
	dRepo := fmt.Sprintf("%s/%s/%s-%s", RegistryHost, c.GetDockerRepo(), c.GetApp(), c.GetSha())
	// Pull docker container
	if pretending() {
		log.Printf("[%s][pretend] deploy with %s @ %s...", c.GetID(), c.GetApp(), c.GetSha())
		log.Printf("[%s][pretend] docker pull %s", c.GetID(), dRepo)
		log.Printf("[%s][pretend] docker run %s", c.GetID(), dRepo)
		c.SetDockerID(fmt.Sprintf("pretend-docker-id-%s", c.GetID()))
	} else {
		log.Printf("[%s] deploy with %s @ %s...", c.GetID(), c.GetApp(), c.GetSha())
		log.Printf("[%s] docker pull %s", c.GetID(), dRepo)
		dockerLock.Lock()
		err := dockerClient.PullImage(docker.PullImageOptions{Repository: dRepo},
			os.Stdout)
		dockerLock.Unlock()
		if err != nil {
			return err
		}

		// make log dir for volume
		err = os.MkdirAll(helper.HostLogDir(c.GetID()), 0755)
		if err != nil {
			return err
		}
		// make config dir for volume
		err = os.MkdirAll(helper.HostConfigDir(c.GetID()), 0755)
		if err != nil {
			return err
		}
		// put config in config dir
		appCfg, err := AppCfgs(c)
		if err != nil {
			return err
		}
		if err := appCfg.Save(helper.HostConfigFile(c.GetID())); err != nil {
			RemoveConfigDir(c)
			return err
		}

		log.Printf("[%s] docker run %s", c.GetID(), dRepo)
		// create docker container
		dCfg, dHostCfg := DockerCfgs(c)
		dockerLock.Lock()
		dCont, err := dockerClient.CreateContainer(docker.CreateContainerOptions{Name: c.GetID()}, dCfg)
		dockerLock.Unlock()
		if err != nil {
			log.Printf("[%s] ERROR: failed to create container: %s", c.GetID(), err.Error())
			return err
		}
		c.SetDockerID(dCont.ID)

		// start docker container
		dockerLock.Lock()
		err = dockerClient.StartContainer(c.GetDockerID(), dHostCfg)
		dockerLock.Unlock()
		if err != nil {
			log.Printf("[%s] ERROR: failed to start container: %s", c.GetID(), err.Error())
			log.Printf("[%s] -- full create response:\n%+v", c.GetID(), dCont)
			log.Printf("[%s] inspecting container for more information...", c.GetID())
			dockerLock.Lock()
			inspCont, ierr := dockerClient.InspectContainer(c.GetDockerID())
			dockerLock.Unlock()
			if ierr != nil {
				log.Printf("[%s] ERROR: failed to inspect container: %s", c.GetID(), ierr.Error())
				return ierr
			}
			log.Printf("[%s] -- inspected container:\n%+v", c.GetID(), inspCont)
			return err
		}

		dockerLock.Lock()
		inspCont, err := dockerClient.InspectContainer(c.GetDockerID())
		dockerLock.Unlock()
		if err != nil {
			log.Printf("[%s] ERROR: failed to inspect container: %s", c.GetID(), err.Error())
			return err
		}
		if inspCont.NetworkSettings == nil {
				log.Printf("[%s] ERROR: failed to get container network settings.")
			return errors.New("Could not get NetworkSettings from docker")
		}
		c.SetIP(inspCont.NetworkSettings.IPAddress)
	}
	return nil
}

func RemoveConfigDir(c types.GenericContainer) error {
	return os.RemoveAll(helper.HostConfigDir(c.GetID()))
}

// Teardown the container. This will kill the docker container but will not free the ports/containers
func Teardown(c types.GenericContainer) error {
	if pretending() {
		log.Printf("[pretend] teardown %s...", c.GetID())
		return nil
	} else {
		log.Printf("teardown %s...", c.GetID())
	}
	defer removeExited()
	dockerLock.Lock()
	err := dockerClient.KillContainer(c.GetDockerID())
	dockerLock.Unlock()
	if err != nil {
		log.Printf("failed to teardown[kill] %s: %v", c.GetID(), err)
		return err
	}
	// TODO do something with log dir
	return RemoveConfigDir(c)
}
