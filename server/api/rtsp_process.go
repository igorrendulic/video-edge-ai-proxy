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
	"strings"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-redis/redis/v7"
)

type rtspProcessHandler struct {
	processManager  *services.ProcessManager
	settingsManager *services.SettingsManager
	rdb             *redis.Client
}

func NewRTSPProcessHandler(rdb *redis.Client, processManager *services.ProcessManager, settingsManager *services.SettingsManager) *rtspProcessHandler {
	return &rtspProcessHandler{
		processManager:  processManager,
		settingsManager: settingsManager,
		rdb:             rdb,
	}
}

func (ph *rtspProcessHandler) StartRTSP(c *gin.Context) {
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

	rtspImageTag := models.CameraTypeToImageTag["rtsp"]
	currentImagesList, err := ph.settingsManager.ListDockerImages(rtspImageTag)
	if err != nil {
		g.Log.Error("failed to list currently available images", err)
		AbortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	err = ph.processManager.Start(&streamProcess, currentImagesList)
	if err != nil {
		g.Log.Warn("failed to start process ", deviceID, err)
		AbortWithError(c, http.StatusConflict, err.Error())
		return
	}
	// publish to chrysalis cloud the change
	utils.PublishToRedis(ph.rdb, deviceID, models.MQTTProcessOperation(models.DeviceOperationAdd), models.ProcessTypeRTSP, nil)

	c.Status(http.StatusOK)
}

// FindUpgrades - checks if each process has an upgradable version available on local disk
func (ph *rtspProcessHandler) FindRTSPUpgrades(c *gin.Context) {

	imageTag := models.CameraTypeToImageTag["rtsp"]

	imageUpgrade, err := ph.settingsManager.ListDockerImages(imageTag)
	if err != nil {
		AbortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	upgrades, err := ph.processManager.FindUpgrades(imageUpgrade)
	if err != nil {
		g.Log.Error("failed finding image upgrades", err)
		AbortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	// publish to chrysalis cloud the change
	for _, upgr := range upgrades {
		utils.PublishToRedis(ph.rdb, upgr.Name, models.MQTTProcessOperation(models.DeviceOperationUpgradeAvailable), models.ProcessTypeRTSP, nil)
	}

	c.JSON(http.StatusOK, upgrades)
}

// UpgradeContainer - upgrades a running container for specific process
func (ph *rtspProcessHandler) UpgradeContainer(c *gin.Context) {

	var process models.StreamProcess
	if err := c.ShouldBindWith(&process, binding.JSON); err != nil {
		g.Log.Warn("missing required fields", err)
		AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if process.ImageTag == "" {
		AbortWithError(c, http.StatusBadRequest, "imagetag is empty on StreamProcess")
		return
	}

	splitted := strings.Split(process.ImageTag, ":")
	if len(splitted) != 2 {
		AbortWithError(c, http.StatusBadRequest, "invalid image. tag (verion) required")
		return
	}
	baseTag := splitted[0]

	newProc, err := ph.processManager.UpgradeRunningContainer(&process, baseTag+":"+process.NewerVersion)
	if err != nil {
		g.Log.Error("failed to upgrade running container", process.Name, process.ImageTag)
		AbortWithError(c, http.StatusConflict, err.Error())
		return
	}
	// publish to chrysalis cloud the change
	utils.PublishToRedis(ph.rdb, process.Name, models.MQTTProcessOperation(models.DeviceOperationAdd), models.ProcessTypeRTSP, nil)

	c.JSON(http.StatusOK, newProc)
}

func (ph *rtspProcessHandler) Stop(c *gin.Context) {
	deviceID := c.Param("name")
	if deviceID == "" {
		AbortWithError(c, http.StatusBadRequest, "required device_id")
		return
	}
	err := ph.processManager.Stop(deviceID, models.PrefixRTSPProcess)
	if err != nil {
		g.Log.Warn("failed to start process ", deviceID, err)
		AbortWithError(c, http.StatusConflict, err.Error())
		return
	}
	// publish to chrysalis cloud the change
	utils.PublishToRedis(ph.rdb, deviceID, models.MQTTProcessOperation(models.DeviceOperationRemove), models.ProcessTypeRTSP, nil)
	c.Status(http.StatusOK)
}

// Info of the specific process
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
