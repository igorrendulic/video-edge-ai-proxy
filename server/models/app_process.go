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

package models

import (
	microModelDocker "github.com/chryscloud/go-microkit-plugins/models/docker"
	"github.com/docker/docker/api/types"
)

const (
	PrefixAppProcess = "/appprocess/"
	RuntimeNvidia    = "nvidia"
)

type AppProcess struct {
	Name                string                       `json:"name"`                            // requrired: custom name of the app
	DockerHubUser       string                       `json:"docker_user"`                     // docker hub user
	DockerhubRepository string                       `json:"docker_repository"`               // dockerhub repository
	EnvVars             []*VarPair                   `json:"env_vars,omitempty"`              // app environment arguments
	ArgsVars            []*VarPair                   `json:"arguments,omitempty"`             // argument parameters
	PortMapping         []*PortMap                   `json:"port_mappings,omitempty"`         // optional: port mappings for the app
	MountFolders        []*VarPair                   `json:"mount,omitempty"`                 // mount folders
	Runtime             string                       `json:"runtime"`                         // (e.g. nvidia)
	DockerHubVersion    string                       `json:"docker_version"`                  // the version of the app from docker hub
	UpgradeAvailable    bool                         `json:"upgrade_available,default:false"` // by default no upgrade available
	NewerVersion        string                       `json:"newer_version,omitempty"`         // if upgrade true the latest version available
	ContainerID         string                       `json:"container_id,omitempty"`          // docker container id
	Status              string                       `json:"status,omitempty"`                // State (running, restarting, ...)
	State               *types.ContainerState        `json:"state,omitempty"`                 // state copies from container
	Logs                *microModelDocker.DockerLogs `json:"logs,omitempty"`                  // logs (error and info)
	Created             int64                        `json:"created,omitempty"`               // unix timestamp in ms when created
	Modified            int64                        `json:"modified,omitempty"`              // last modificadation date, epoch in ms
}

type VarPair struct {
	Name  string `json:"name"`  // name of the variable
	Value string `json:"value"` // value of the variable
}

type PortMap struct {
	Exposed int `json:"exposed"` // exposed port
	MapTo   int `json:"map_to"`  // to which internal port exposed port maps to
}

func ValidateApp(app *AppProcess) error {
	if app.DockerHubUser == "" || app.DockerHubVersion == "" || app.DockerhubRepository == "" {
		return ErrMissingInputParameters
	}
	if len(app.Name) < 3 {
		return ErrStringTooShort
	}
	return nil
}
