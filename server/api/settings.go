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
	"encoding/json"
	"net/http"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
	"github.com/dgraph-io/badger/v2"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-resty/resty/v2"
)

type settingsHandler struct {
	settingsManager *services.SettingsManager
	apiClient       *resty.Client
}

func NewSettingsHandler(settingsManager *services.SettingsManager) *settingsHandler {
	return &settingsHandler{
		settingsManager: settingsManager,
		apiClient:       resty.New(),
	}
}

// Get settings
func (sh *settingsHandler) Get(c *gin.Context) {
	s, err := sh.settingsManager.Get()
	if err != nil {
		g.Log.Error("failed to retrieve settings", err)
		AbortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, s)
}

// Overwrite settings
func (sh *settingsHandler) Overwrite(c *gin.Context) {
	var settings models.Settings
	if err := c.ShouldBindWith(&settings, binding.JSON); err != nil {
		g.Log.Warn("missing required fields", err)
		AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	// get settings first, because we might have gatewayID and registryID already stored
	existingSettings, eErr := sh.settingsManager.Get()
	if eErr != nil {
		if eErr != badger.ErrKeyNotFound {
			g.Log.Error("failed to retrieve existing settings from datastore", eErr)
			AbortWithError(c, http.StatusInternalServerError, "Failed to retrieve settings from datastore")
			return
		}
	}

	queryParams := ""
	if existingSettings != nil {
		// queryParams += "?registryId=" + existingSettings.RegistryID + "&gatewayId=" + existingSettings.GatewayID
	}

	// validate settings with the Chrysalis Cloud
	resp, apiErr := utils.CallAPIWithBody(sh.apiClient, "GET", g.Conf.API.Endpoint+"/api/v1/edge/credentials"+queryParams, "", settings.EdgeKey, settings.EdgeSecret)
	if apiErr != nil {
		g.Log.Error("Failed to validate credentials with chrys cloud", apiErr)
		AbortWithError(c, http.StatusUnauthorized, "Failed to validate credentials with Chryscloud")
		return
	}
	var cloudResponse models.EdgeConnectCredentials
	mErr := json.Unmarshal(resp, &cloudResponse)
	if mErr != nil {
		g.Log.Error("failed to unmarshal response from Chryscloud", mErr)
		AbortWithError(c, http.StatusExpectationFailed, "Failed to unmarshal response from Chryscloud. Please upgrade Chrysalis Edge Proxy to latest version")
		return
	}
	settings.ProjectID = cloudResponse.ProjectID
	settings.RegistryID = cloudResponse.RegistryID
	settings.GatewayID = cloudResponse.GatewayID
	settings.Region = cloudResponse.Region
	settings.PrivateRSAKey = cloudResponse.PrivateKeyPem

	err := sh.settingsManager.Overwrite(&settings)
	if err != nil {
		AbortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusAccepted)
}

// DockerImagesLocally finds images that correspond with the image_name and returns has_downloadded or maybe if upgraded needed (newer version available)
func (sh *settingsHandler) DockerImagesLocally(c *gin.Context) {

	tagName := c.Query("tag")
	if tagName == "" {
		AbortWithError(c, http.StatusNotFound, "not found")
		return
	}

	images, err := sh.settingsManager.ListDockerImages(tagName)
	if err != nil {
		g.Log.Error("failed to list docker image", err)
		AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, images)
}

// DockerPullImage pulls the docker image from the DockerHub to local machine
func (sh *settingsHandler) DockerPullImage(c *gin.Context) {
	name := c.Query("tag")
	version := c.Query("version")

	if name == "" || version == "" {
		AbortWithError(c, http.StatusNotFound, "not found")
		return
	}

	pullResp, err := sh.settingsManager.PullDockerImage(name, version)
	if err != nil {
		g.Log.Error("failed to pull docker image", err)
		AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, pullResp)
}

func (sh *settingsHandler) SetCurrentCameraDockerImageVersion(c *gin.Context) {
	tagWithVersion := c.Query("tag")
	if tagWithVersion == "" {
		AbortWithError(c, http.StatusBadRequest, "tag with version missing")
		return
	}

}
