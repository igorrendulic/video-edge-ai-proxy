// Copyright 2020 Wearless Tech Inc All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/chryscloud/go-microkit-plugins/docker"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	dockerErrors "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/go-redis/redis/v7"
)

// ProcessManager - start, stop of docker containers
type AppProcessManager struct {
	storage *Storage
	rdb     *redis.Client
}

func NewAppManager(storage *Storage, rdb *redis.Client) *AppProcessManager {
	return &AppProcessManager{
		storage: storage,
		rdb:     rdb,
	}
}

// Install - installs the new app
func (am *AppProcessManager) Install(app *models.AppProcess) (*models.AppProcess, error) {

	// installation process
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))

	fl := filters.NewArgs()
	pruneReport, pruneErr := cl.ContainersPrune(fl)
	if pruneErr != nil {
		g.Log.Error("container prunning fialed", pruneErr)
		return nil, pruneErr
	}
	g.Log.Info("app prune successfull. Report and space reclaimed", pruneReport.ContainersDeleted, pruneReport.SpaceReclaimed)

	// expose desired ports mappings if any
	portMap := nat.PortMap{}
	portSet := nat.PortSet{}
	if len(app.PortMapping) > 0 {

		for _, pm := range app.PortMapping {
			exposedPort := strconv.Itoa(pm.Exposed)
			mapsTo := pm.MapTo

			mapsToPort := strconv.Itoa(mapsTo) + "/tcp"
			portSet[nat.Port(mapsToPort)] = struct{}{}
			portMap[nat.Port(mapsToPort)] = []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: exposedPort}}
		}
	}

	// prepare host configuration
	hostConfig := &container.HostConfig{
		LogConfig: container.LogConfig{
			Type:   "json-file",
			Config: map[string]string{"max-file": "3", "max-size": "3M"},
		},
		RestartPolicy: container.RestartPolicy{Name: "on-failure", MaximumRetryCount: 10},
		Resources: container.Resources{
			CPUShares: 1024, // equal weight to all containers. check here the docs here:  https://docs.docker.com/config/containers/resource_constraints/
		},
		NetworkMode: container.NetworkMode("chrysnet"),
	}
	if app.Runtime == models.RuntimeNvidia {
		hostConfig.Runtime = models.RuntimeNvidia
		capabilites := [][]string{{"gpu", "nvidia", "compute"}}
		hostConfig.DeviceRequests = []container.DeviceRequest{{Driver: models.RuntimeNvidia, Capabilities: capabilites, Count: -1}}
	}
	if len(portMap) > 0 {
		hostConfig.PortBindings = portMap
	}

	// prepare mounted folders if any
	if len(app.MountFolders) > 0 {
		mounts := make([]mount.Mount, 0)
		for _, mnt := range app.MountFolders {
			mount := mount.Mount{
				Type:     mount.TypeBind,
				Source:   mnt.Name,
				Target:   mnt.Value,
				ReadOnly: false,
			}
			mounts = append(mounts, mount)
		}
		hostConfig.Mounts = mounts
	}

	// prepare environment variables if any
	envVars := []string{}
	if len(app.EnvVars) > 0 {
		for _, env := range app.EnvVars {
			envVars = append(envVars, env.Name+"="+env.Value)
		}
	}

	envVars = append(envVars, "PYTHONUNBUFFERED=0") // for output to console

	// prepare image tag
	imageTag := app.DockerHubUser + "/" + app.DockerhubRepository + ":" + app.DockerHubVersion

	// preapre container configuration
	containerConf := &container.Config{
		Image: imageTag,
		Env:   envVars,
	}
	if len(portSet) > 0 {
		containerConf.ExposedPorts = portSet
	}
	_, ccErr := cl.ContainerCreate(strings.ToLower(app.Name), containerConf, hostConfig, nil)

	if ccErr != nil {
		g.Log.Error("failed to create container ", app.Name, ccErr)
		return nil, ccErr
	}

	err := cl.ContainerStart(app.Name)
	if err != nil {
		g.Log.Error("failed to start container", app.Name, err)
		return nil, err
	}

	app.Status = models.ProcessStatusRunning
	app.Created = time.Now().Unix() * 1000
	app.Modified = time.Now().Unix() * 1000

	obj, err := json.Marshal(app)
	if err != nil {
		g.Log.Error("failed to marshal process json", err)
		return nil, err
	}

	err = am.storage.Put(models.PrefixAppProcess, app.Name, obj)
	if err != nil {
		g.Log.Error("failed to store process into datastore", err)
		return nil, err
	}

	return app, nil
}

func (am *AppProcessManager) ListApps() ([]*models.AppProcess, error) {
	objects, err := am.storage.List(models.PrefixAppProcess)
	if err != nil {
		g.Log.Error("failed to list devices", err)
		return nil, err
	}
	processes := make([]*models.AppProcess, 0)
	for _, v := range objects {
		var process models.AppProcess
		dErr := json.Unmarshal(v, &process)
		if dErr != nil {
			g.Log.Error("failed to unamrshal object", err)
			return nil, err
		}
		processes = append(processes, &process)
	}
	deleteProcesses := make([]*models.AppProcess, 0)
	cleanProcesses := make([]*models.AppProcess, 0)
	// clean up and update the list
	for _, proc := range processes {
		info, err := am.Info(proc.Name)
		if err != nil {
			g.Log.Warn("failed to load process", err)
			if err == models.ErrProcessNotFound {
				// remove from the list and datastore
				deleteProcesses = append(deleteProcesses, proc)
				continue
			}
			g.Log.Error("failed to get process info", err)
			return nil, err
		}
		cleanProcesses = append(cleanProcesses, info)
	}
	if len(deleteProcesses) > 0 {
		for _, proc := range deleteProcesses {
			err := am.storage.Del(models.PrefixRTSPProcess, proc.Name)
			if err != nil {
				g.Log.Error("failed to delete process with name", proc.Name, err)
				return nil, err
			}
		}
	}
	return cleanProcesses, nil
}

func (am *AppProcessManager) Info(appName string) (*models.AppProcess, error) {
	// Info - return information on the streaming docker container (it also updates the process status)
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))
	container, err := cl.ContainerGet(appName)
	if err != nil {
		if dockerErrors.IsErrNotFound(err) {
			g.Log.Info("container not found to be stopeed", err)
			return nil, models.ErrProcessNotFound
		}
		g.Log.Error("failed to retrieve container", err)
		return nil, err
	}

	// max 100 lines of logs
	logs, err := cl.ContainerLogs(container.ID, 100, time.Unix(0, 0))
	if err != nil {
		g.Log.Error("failed to retrieve container logs", err)
		return nil, err
	}

	sp, err := am.storage.Get(models.PrefixAppProcess, appName)
	if err != nil {
		g.Log.Error("failed to find device with name", appName, err)
		return nil, models.ErrProcessNotFoundDatastore
	}
	var status models.AppProcess
	err = json.Unmarshal(sp, &status)
	if err != nil {
		g.Log.Error("failed to unmarshal stored process ", err)
		return nil, err
	}
	status.ContainerID = container.ID
	if container != nil {
		status.State = container.State
		status.Status = container.State.Status
	} else {
		status.Status = "unknown"
	}
	status.Logs = logs
	status.Modified = time.Now().Unix() * 1000

	b, err := json.Marshal(&status)
	if err != nil {
		g.Log.Error("failed to marshal process", err)
		return nil, err
	}
	err = am.storage.Put(models.PrefixAppProcess, status.Name, b)
	if err != nil {
		g.Log.Error("failed to store process after info", err)
		return nil, err
	}

	return &status, nil
}
