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
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type settingsHandler struct {
	settingsManager *services.SettingsManager
}

func NewSettingsHandler(settingsManager *services.SettingsManager) *settingsHandler {
	return &settingsHandler{
		settingsManager: settingsManager,
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

	newSett, err := sh.settingsManager.Overwrite(&settings)
	if err != nil {
		AbortWithError(c, http.StatusExpectationFailed, "Failed to store settings. Make sure you have an internet connection and try again")
		return
	}
	c.JSON(http.StatusOK, newSett)
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
