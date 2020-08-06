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

package api

import (
	"crypto/md5"
	"fmt"
	"net/http"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type rtspProcessHandler struct {
	processManager *services.ProcessManager
}

func NewRTSPProcessHandler(processManager *services.ProcessManager) *rtspProcessHandler {
	return &rtspProcessHandler{
		processManager: processManager,
	}
}

func (ph *rtspProcessHandler) Start(c *gin.Context) {
	var streamProcess models.StreamProcess
	if err := c.ShouldBindWith(&streamProcess, binding.JSON); err != nil {
		g.Log.Warn("missing required fields", err)
		AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if streamProcess.RTSPEndpoint == "" {
		AbortWithError(c, http.StatusBadRequest, "RTP endpoint required")
		return
	}
	deviceID := streamProcess.Name
	if streamProcess.Name == "" {
		hash := fmt.Sprintf("%x", md5.Sum([]byte(streamProcess.RTSPEndpoint)))
		deviceID = hash
	}
	streamProcess.RTMPStreamStatus = &models.RTMPStreamStatus{
		Storing:   false,
		Streaming: true,
	}

	err := ph.processManager.Start(&streamProcess)
	if err != nil {
		g.Log.Warn("failed to start process ", deviceID, err)
		AbortWithError(c, http.StatusConflict, err.Error())
		return
	}
	c.Status(http.StatusOK)
}

func (ph *rtspProcessHandler) Stop(c *gin.Context) {
	deviceID := c.Param("name")
	if deviceID == "" {
		AbortWithError(c, http.StatusBadRequest, "required device_id")
		return
	}
	err := ph.processManager.Stop(deviceID)
	if err != nil {
		g.Log.Warn("failed to start process ", deviceID, err)
		AbortWithError(c, http.StatusConflict, err.Error())
		return
	}
	c.Status(http.StatusOK)
}

func (ph *rtspProcessHandler) Info(c *gin.Context) {
	deviceID := c.Param("name")
	if deviceID == "" {
		AbortWithError(c, http.StatusBadRequest, "required device_id")
		return
	}
	info, err := ph.processManager.Info(deviceID)
	if err != nil {
		AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, info)
}

func (ph *rtspProcessHandler) List(c *gin.Context) {
	processes, err := ph.processManager.List()
	if err != nil {
		AbortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, processes)
}
