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
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/chryscloud/go-microkit-plugins/docker"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerErrors "github.com/docker/docker/client"
)

// ProcessManager - start, stop of docker containers
type ProcessManager struct {
	storage *Storage
}

func NewProcessManager(storage *Storage) *ProcessManager {
	return &ProcessManager{
		storage: storage,
	}
}

// Start - starts the docker container with rtsp, device_id and possibly rtmp environment variables.
// Restarts always when something goes wrong within the streaming process
func (pm *ProcessManager) Start(process *models.StreamProcess) error {

	if process.Name == "" || process.RTSPEndpoint == "" {
		return errors.New("required parameters missing")
	}
	// default docker image (must be pulled manually for now)
	if process.ImageTag == "" {
		process.ImageTag = "chryscloud/chrysedgeproxy:0.0.1"
	}

	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))

	fl := filters.NewArgs()
	pruneReport, pruneErr := cl.ContainersPrune(fl)
	if pruneErr != nil {
		g.Log.Error("container prunning fialed", pruneErr)
		return pruneErr
	}
	g.Log.Info("prune successfull. Report and space reclaimed:", pruneReport.ContainersDeleted, pruneReport.SpaceReclaimed)

	hostConfig := &container.HostConfig{
		// PortBindings: mappingPorts,
		LogConfig: container.LogConfig{
			Type:   "json-file",
			Config: map[string]string{"max-file": "3", "max-size": "3M"},
		},
		RestartPolicy: container.RestartPolicy{Name: "always"},
		Resources: container.Resources{
			CPUShares: 1024, // equal weight to all containers. check here the docs here:  https://docs.docker.com/config/containers/resource_constraints/
		},
		NetworkMode: container.NetworkMode("chrysnet"),
	}

	envVars := []string{"rtsp_endpoint=" + process.RTSPEndpoint, "device_id=" + process.Name}
	if process.RTMPEndpoint != "" {
		envVars = append(envVars, "rtmp_endpoint="+process.RTMPEndpoint)
	}
	envVars = append(envVars, "PYTHONUNBUFFERED=0")

	_, err := cl.ContainerCreate(strings.ToLower(process.Name), &container.Config{
		Image: process.ImageTag,
		Env:   envVars,
	}, hostConfig, nil)

	err = cl.ContainerStart(process.Name)
	if err != nil {
		g.Log.Error("failed to start container", process.Name, err)
		return err
	}

	process.Status = "running"
	process.Created = time.Now().Unix() * 1000
	obj, err := json.Marshal(process)
	if err != nil {
		g.Log.Error("failed to marshal process json", err)
		return err
	}

	err = pm.storage.Put(models.PrefixRTSPProcess, process.Name, obj)
	if err != nil {
		g.Log.Error("failed to store process into datastore", err)
		return err
	}

	return nil
}

// Stop - stops the docker container by the name of deviceID and removed from local datastore
func (pm *ProcessManager) Stop(deviceID string) error {
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))
	container, err := cl.ContainerGet(deviceID)
	if err != nil {
		if dockerErrors.IsErrContainerNotFound(err) {
			g.Log.Info("container not found to be stopeed", err)
			return err
		}
	}

	// waits up to 10 minutes to stop the container, otherwise kills after 30 seconds
	killAfer := time.Second * 5
	err = cl.ContainerStop(container.ID, &killAfer)
	if err != nil {
		if dockerErrors.IsErrNotFound(err) {
			g.Log.Warn("container doesn't exist. probably stopped before", err)
			return nil
		}
	}

	err = pm.storage.Del(models.PrefixRTSPProcess, deviceID)
	if err != nil {
		g.Log.Error("Failed to delete rtsp proces", err)
		return err
	}

	fl := filters.NewArgs()
	pruneReport, pruneErr := cl.ContainersPrune(fl)
	if pruneErr != nil {
		g.Log.Error("container prunning fialed", pruneErr)
		return pruneErr
	}
	g.Log.Info("prune successfull. Report and space reclaimed:", pruneReport.ContainersDeleted, pruneReport.SpaceReclaimed)

	return nil
}

// ListStream - GRPC method for list all streams (doesn't alter the actual processes)
func (pm *ProcessManager) ListStream(ctx context.Context, found func(process *models.StreamProcess) error) error {
	objects, err := pm.storage.List(models.PrefixRTSPProcess)
	if err != nil {
		g.Log.Error("failed to list devices", err)
		return err
	}
	processes := make([]*models.StreamProcess, 0)
	for _, v := range objects {
		var process models.StreamProcess
		dErr := json.Unmarshal(v, &process)
		if dErr != nil {
			g.Log.Error("failed to unamrshal object", err)
			return err
		}
		processes = append(processes, &process)
	}
	// clean up and update the list
	for _, proc := range processes {
		if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
			g.Log.Warn("context is cancelled")
			return nil
		}
		info, err := pm.Info(proc.Name)
		if err != nil {
			g.Log.Warn("failed to load process", err)
			if err == ErrProcessNotFound {
				continue
			}
			g.Log.Error("failed to get process info", err)
			return err
		}
		err = found(info)
		if err != nil {
			g.Log.Error("failed to return found process", err)
			return err
		}
	}
	if err != nil {
		g.Log.Error("unexpcted error on unidirectional process info stream", err)
		return err
	}
	return nil
}

// List - listing all of the process in any status (also augments the list based on current processes)
func (pm *ProcessManager) List() ([]*models.StreamProcess, error) {
	objects, err := pm.storage.List(models.PrefixRTSPProcess)
	if err != nil {
		g.Log.Error("failed to list devices", err)
		return nil, err
	}
	processes := make([]*models.StreamProcess, 0)
	for _, v := range objects {
		var process models.StreamProcess
		dErr := json.Unmarshal(v, &process)
		if dErr != nil {
			g.Log.Error("failed to unamrshal object", err)
			return nil, err
		}
		processes = append(processes, &process)
	}

	deleteProcesses := make([]*models.StreamProcess, 0)
	cleanProcesses := make([]*models.StreamProcess, 0)
	// clean up and update the list
	for _, proc := range processes {
		info, err := pm.Info(proc.Name)
		if err != nil {
			g.Log.Warn("failed to load process", err)
			if err == ErrProcessNotFound {
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
			err := pm.storage.Del(models.PrefixRTSPProcess, proc.Name)
			if err != nil {
				g.Log.Error("failed to delete process with name", proc.Name, err)
				return nil, err
			}
		}
	}
	return cleanProcesses, nil
}

// Info - return information on the streaming docker container (it also updates the process status)
func (pm *ProcessManager) Info(deviceID string) (*models.StreamProcess, error) {
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))
	container, err := cl.ContainerGet(deviceID)
	if err != nil {
		if dockerErrors.IsErrContainerNotFound(err) {
			g.Log.Info("container not found to be stopeed", err)
			return nil, ErrProcessNotFound
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

	sp, err := pm.storage.Get(models.PrefixRTSPProcess, deviceID)
	if err != nil {
		g.Log.Error("failed to find device with name", deviceID, err)
		return nil, ErrProcessNotFoundDatastore
	}
	var status models.StreamProcess
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
	err = pm.storage.Put(models.PrefixRTSPProcess, status.Name, b)
	if err != nil {
		g.Log.Error("failed to store process after info", err)
		return nil, err
	}

	return &status, nil
}

// UpdateProcessInfo - start and stop information propagated into redis and state stored into datastore
func (pm *ProcessManager) UpdateProcessInfo(stream *models.StreamProcess) (*models.StreamProcess, error) {

	stream.Modified = time.Now().Unix() * 1000 // miliseconds

	b, err := json.Marshal(stream)
	if err != nil {
		g.Log.Error("failed to marshal process", err)
		return nil, err
	}
	err = pm.storage.Put(models.PrefixRTSPProcess, stream.Name, b)
	if err != nil {
		g.Log.Error("failed to store process after info", err)
		return nil, err
	}

	// TODO: add to redis

	return stream, nil
}
