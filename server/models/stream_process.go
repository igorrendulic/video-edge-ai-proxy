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
	PrefixRTSPProcess = "/rtspprocess/"

	// docker process status mapping
	ProcessStatusCreated    = "created"
	ProcessStatusRestarting = "restarting"
	ProcessStatusRunning    = "running"
	ProcessStatusRemoving   = "removing"
	ProcessStatusPaused     = "paused"
	ProcessStatusExited     = "exited"
	ProcessStatusDead       = "dead"
	// added custom status for chrysalis cloud
	ProcessStatusFailed     = "failed"
	ProcessStatusInProgress = "in-progress"
)

type StreamProcess struct {
	Name             string                       `json:"name,omitempty"`                   // name of the streaming process
	ImageTag         string                       `json:"image_tag,omitempty"`              // imagetag (default is: chryscloud/chrysedgeproxy:latest)
	RTSPEndpoint     string                       `json:"rtsp_endpoint" binding:"required"` // rtsp_endpoint (always required for RTSP streaming)
	RTMPEndpoint     string                       `json:"rtmp_endpoint,omitempty"`          // RTMP endpoint (optional Chrysalis RTMP endpoint)
	ContainerID      string                       `json:"container_id,omitempty"`           // docker container id
	Status           string                       `json:"status,omitempty"`                 // State (running, restarting, ...)
	State            *types.ContainerState        `json:"state,omitempty"`                  // state copies from container
	Logs             *microModelDocker.DockerLogs `json:"logs,omitempty"`                   // logs (error and info)
	Created          int64                        `json:"created,omitempty"`                // unix timestamp in ms when created
	Modified         int64                        `json:"modified,omitempty"`               // last modificadation date, epoch in ms
	RTMPStreamStatus *RTMPStreamStatus            `json:"rtmp_stream_status,omitempty"`     // info on if stream is being proxied and if storage set
	UpgradeAvailable bool                         `json:"upgrade_available,default:false"`  // by default no upgrade available
	NewerVersion     string                       `json:"newer_version,omitempty"`          // if upgrade true the latest version available
}

type RTMPStreamStatus struct {
	Streaming bool `json:"streaming"`
	Storing   bool `json:"storing"`
}
