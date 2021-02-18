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
	"net/http"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-redis/redis/v7"
)

type appProcessHandler struct {
	processManager  *services.ProcessManager
	settingsManager *services.SettingsManager
	appManager      *services.AppProcessManager
	rdb             *redis.Client
}

func NewAppProcessHandler(rdb *redis.Client, appManager *services.AppProcessManager, processManager *services.ProcessManager, settingsManager *services.SettingsManager) *appProcessHandler {
	return &appProcessHandler{
		processManager:  processManager,
		settingsManager: settingsManager,
		appManager:      appManager,
		rdb:             rdb,
	}
}

// Installs the app
func (aph *appProcessHandler) InstallApp(c *gin.Context) {
	var apProcess models.AppProcess
	if err := c.ShouldBindWith(&apProcess, binding.JSON); err != nil {
		g.Log.Warn("missing required fields", err)
		AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// validation of input
	err := models.ValidateApp(&apProcess)
	if err != nil {
		if err == models.ErrMissingInputParameters {
			AbortWithError(c, http.StatusBadRequest, "missing required input parameters")
			return
		}
		if err == models.ErrStringTooShort {
			AbortWithError(c, http.StatusBadRequest, "name must have at least 3 characters")
			return
		}
		g.Log.Error("input validation failed", err)
		AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// always update image
	_, err = aph.settingsManager.PullDockerImage(apProcess.DockerHubUser+"/"+apProcess.DockerhubRepository, apProcess.DockerHubVersion)
	if err != nil {
		g.Log.Error("failed to pull image from dockerhub", err)
		AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	_, err = aph.appManager.Install(&apProcess)
	if err != nil {
		AbortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

}

func (aph *appProcessHandler) RemoveApp(c *gin.Context) {
	deviceID := c.Param("name")
	if deviceID == "" {
		AbortWithError(c, http.StatusBadRequest, "required device_id")
		return
	}
	err := aph.processManager.Stop(deviceID, models.PrefixAppProcess)
	if err != nil {
		g.Log.Warn("failed to start process ", deviceID, err)
		AbortWithError(c, http.StatusConflict, err.Error())
		return
	}
	// publish to chrysalis cloud the change
	utils.PublishToRedis(aph.rdb, deviceID, models.MQTTProcessOperation(models.DeviceOperationRemove), models.ProcessTypeApplication, nil)
	c.Status(http.StatusOK)
}

func (aph *appProcessHandler) ListApps(c *gin.Context) {
	apps, err := aph.appManager.ListApps()
	if err != nil {
		AbortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, apps)
}

// Info of the specific process
func (aph *appProcessHandler) Info(c *gin.Context) {
	deviceID := c.Param("name")
	if deviceID == "" {
		AbortWithError(c, http.StatusBadRequest, "required device_id")
		return
	}
	info, err := aph.appManager.Info(deviceID)
	if err != nil {
		AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, info)
}
