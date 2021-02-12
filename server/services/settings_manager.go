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
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chryscloud/go-microkit-plugins/docker"
	dockerhub "github.com/chryscloud/go-microkit-plugins/dockerhub"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
	"github.com/dgraph-io/badger/v2"
	"github.com/docker/docker/api/types"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-version"
)

// SettingsManager - various settings for the edge
type SettingsManager struct {
	storage             *Storage
	current_edge_key    string
	current_edge_secret string
	mux                 *sync.RWMutex
	apiClient           *resty.Client
}

func NewSettingsManager(storage *Storage) *SettingsManager {
	return &SettingsManager{
		storage:   storage,
		mux:       &sync.RWMutex{},
		apiClient: resty.New(),
	}
}

func (sm *SettingsManager) GetCurrentEdgeKeyAndSecret() (string, string, error) {
	if sm.current_edge_key == "" || sm.current_edge_secret == "" {
		settings, err := sm.getDefault()
		if err != nil {
			if err != badger.ErrKeyNotFound {
				g.Log.Error("failed to query for current edge api key and secret", err)
			}
			return "", "", err
		}
		sm.mux.Lock()
		defer sm.mux.Unlock()
		sm.current_edge_key = settings.EdgeKey
		sm.current_edge_secret = settings.EdgeSecret
	}
	return sm.current_edge_key, sm.current_edge_secret, nil
}

// Used on systm start, calling cloud to connect to (and refresh possible keys, cert, ...)
func (sm *SettingsManager) UpdateEdgeRegistrationToCloud() error {
	defaultSettings, err := sm.getDefault()
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil
		}
		g.Log.Error("failed to query for settings in datastore", err)
		return err
	}

	// if not connected to edge there is nothing to do
	if defaultSettings.EdgeKey == "" || defaultSettings.EdgeSecret == "" {
		return nil
	}

	// get system info
	sysInfo, err := sm.GetSystemInfo()
	if err != nil {
		g.Log.Error("failed to retrieve system info", err)
		return err
	}

	_, err = sm.updateSettingsWithMQTTCredentials(sysInfo, defaultSettings)
	return err
}

// getDefault - retrieves settings if exist, otherwise creates new empty settings
func (sm *SettingsManager) getDefault() (*models.Settings, error) {
	// check if settings already exist
	settingsBytes, err := sm.storage.Get(models.PrefixSettingsKey, models.SettingDefaultKey)
	if err != nil {
		if err != badger.ErrKeyNotFound {
			g.Log.Error("failed to retrieve settings", err)
			return nil, err
		}
	}

	var settings models.Settings
	if settingsBytes == nil {
		settings = models.Settings{
			Name: models.SettingDefaultKey,
		}
	} else {
		unmErr := json.Unmarshal(settingsBytes, &settings)
		if unmErr != nil {
			g.Log.Error("failed to unmarshal settings", unmErr)
			return nil, unmErr
		}
	}
	sm.mux.Lock()
	defer sm.mux.Unlock()
	if settings.EdgeKey != "" {
		sm.current_edge_key = settings.EdgeKey
	}
	if settings.EdgeSecret != "" {
		sm.current_edge_secret = settings.EdgeSecret
	}
	return &settings, nil
}

// Overwrite always overwrites the complete settings
func (sm *SettingsManager) Overwrite(settings *models.Settings) (*models.Settings, error) {
	existingSettings, _ := sm.getDefault()

	// get system info
	sysInfo, err := sm.GetSystemInfo()
	if err != nil {
		g.Log.Error("failed to retrieve system info", err)
		return nil, err
	}

	if existingSettings != nil {
		sysInfo.GatewayID = existingSettings.GatewayID
		sysInfo.RegistryID = existingSettings.RegistryID
	}

	updSettings, err := sm.updateSettingsWithMQTTCredentials(sysInfo, settings)
	if err != nil {
		g.Log.Error("failed to update settings", err)
		return nil, err
	}

	return updSettings, nil
}

func (sm *SettingsManager) updateSettingsWithMQTTCredentials(sysInfo *models.SystemInfo, settings *models.Settings) (*models.Settings, error) {
	// validate settings with the Chrysalis Cloud

	if settings.GatewayID != "" && settings.RegistryID != "" {
		sysInfo.GatewayID = settings.GatewayID
		sysInfo.RegistryID = settings.RegistryID
	}

	resp, apiErr := utils.CallAPIWithBody(sm.apiClient, "POST", g.Conf.API.Endpoint+"/api/v1/edge/credentials", sysInfo, settings.EdgeKey, settings.EdgeSecret)
	if apiErr != nil {
		g.Log.Error("Failed to validate credentials with chrys cloud", apiErr)
		// AbortWithError(c, http.StatusUnauthorized, "Failed to validate credentials with Chryscloud")
		return nil, apiErr
	}
	var cloudResponse models.EdgeConnectCredentials
	mErr := json.Unmarshal(resp, &cloudResponse)
	if mErr != nil {
		g.Log.Error("failed to unmarshal response from Chryscloud", mErr)
		// AbortWithError(c, http.StatusExpectationFailed, "Failed to unmarshal response from Chryscloud. Please upgrade Chrysalis Edge Proxy to latest version")
		return nil, mErr
	}
	settings.ProjectID = cloudResponse.ProjectID
	settings.RegistryID = cloudResponse.RegistryID
	settings.GatewayID = cloudResponse.GatewayID
	settings.Region = cloudResponse.Region
	settings.PrivateRSAKey = cloudResponse.PrivateKeyPem

	// curently only edgekey setting
	settings.ProjectID = cloudResponse.ProjectID
	settings.GatewayID = cloudResponse.GatewayID
	settings.PrivateRSAKey = cloudResponse.PrivateKeyPem
	settings.Region = cloudResponse.Region
	settings.RegistryID = cloudResponse.RegistryID

	if settings.Created < 0 {
		settings.Created = time.Now().Unix() * 1000
	}
	settings.Modified = time.Now().Unix() * 1000

	settingsBytes, err := json.Marshal(settings)
	if err != nil {
		g.Log.Error("failed to marshal settings", err)
		return nil, err
	}
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.current_edge_key = settings.EdgeKey
	sm.current_edge_secret = settings.EdgeSecret
	newSettingsErr := sm.storage.Put(models.PrefixSettingsKey, settings.Name, settingsBytes)

	return settings, newSettingsErr
}

// Get settings from datastore
func (sm *SettingsManager) Get() (*models.Settings, error) {
	return sm.getDefault()
}

func (sm *SettingsManager) ListLocalDockerImages() ([]types.ImageSummary, error) {
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))
	images, err := cl.ImagesList()
	if err != nil {
		return nil, err
	}
	return images, nil
}

// ListDockerImages - listing local docker images based on tag name and checking if there is a newer version available
func (sm *SettingsManager) ListDockerImages(nameTag string) (*models.ImageUpgrade, error) {
	// cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))
	images, err := sm.ListLocalDockerImages()
	if err != nil {
		return nil, err
	}

	var options dockerhub.Option
	hubCl := dockerhub.NewClient(options)

	remoteTags, err := hubCl.Tags(nameTag)
	if err != nil {
		g.Log.Error("failed to retrieve remote tags", err)
		return nil, err
	}

	// getting local tags
	localTags := make([]string, 0)

	for _, img := range images {
		tags := img.RepoTags
		for _, tag := range tags {
			if strings.HasPrefix(tag, nameTag) {
				// get the tag version
				splitted := strings.Split(tag, ":")
				if len(splitted) == 2 {
					v := strings.Trim(splitted[1], "")
					if v != "latest" { // ignore latest versions
						localTags = append(localTags, v)
					}
				}
			}
		}
	}

	highestRemoteTagVersion := ""
	highestLocalTagVersion := ""
	highestRemoteVersion := sm.findHighestVersion(remoteTags)
	highestLocalVersion := sm.findHighestVersion(localTags)

	hasUpgrade := false
	if highestLocalVersion != nil && highestRemoteVersion != nil {
		if highestLocalVersion.LessThan(highestRemoteVersion) {
			hasUpgrade = true
		}
	}
	if highestRemoteVersion != nil {
		highestRemoteTagVersion = highestRemoteVersion.Original()
	}
	if highestLocalVersion != nil {
		highestLocalTagVersion = highestLocalVersion.Original()
	}

	camType := "unknown"
	if ct, ok := models.ImageTagVersionToCameraType[nameTag]; ok {
		camType = ct
	}

	resp := &models.ImageUpgrade{
		HasImage:             len(localTags) > 0,
		HasUpgrade:           hasUpgrade,
		Name:                 nameTag,
		HighestRemoteVersion: highestRemoteTagVersion,
		CurrentVersion:       highestLocalTagVersion,
		CameraType:           camType,
	}

	return resp, nil
}

// PullDockerImage - pull docker image from dockerhub
func (sm *SettingsManager) PullDockerImage(name, version string) (*models.PullDockerResponse, error) {
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))
	resp, err := cl.ImagePullDockerHub(name, version, "", "")
	if err != nil {
		g.Log.Error("failed to pull image from dockerhub", name, version, err)
		return nil, err
	}

	fmt.Printf("%v\n", resp)
	g.Log.Info(resp)
	camType := "unknown"
	if ct, ok := models.ImageTagVersionToCameraType[name]; ok {
		camType = ct
	}

	settingTag := &models.SettingDockerTagVersion{
		CameraType: camType,
		Tag:        name,
		Version:    version,
	}
	settingsTagBytes, err := json.Marshal(settingTag)
	if err != nil {
		g.Log.Error("failed to marshal settings", err)
		return nil, err
	}

	sErr := sm.storage.Put(models.PrefixSettingsDockerTagVersions, camType, settingsTagBytes)
	if sErr != nil {
		g.Log.Error("failed to store latest docker tag version", sErr)
		return nil, sErr
	}

	response := &models.PullDockerResponse{
		Response: resp,
	}

	return response, nil
}

// findHighestVersion - finding the highest version from the list of tag:version strings
func (sm *SettingsManager) findHighestVersion(versionsRaw []string) *version.Version {
	if len(versionsRaw) == 0 {
		return nil
	}
	versions := make([]*version.Version, 0)
	for _, raw := range versionsRaw {
		v, _ := version.NewVersion(raw)
		if v != nil {
			versions = append(versions, v)
		}
	}
	sort.Sort(version.Collection(versions))
	if len(versions) > 0 {
		return versions[len(versions)-1]
	}
	return nil
}

// getting dockers host system info
func (sm *SettingsManager) GetSystemInfo() (*models.SystemInfo, error) {
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))

	sys, _, err := cl.SystemWideInfo()
	if err != nil {
		g.Log.Error("Failed to get host system info", err)
		return nil, err
	}
	systemInfo := &models.SystemInfo{
		Architecture:  sys.Architecture,
		NCPUs:         sys.NCPU,
		TotalMemory:   sys.MemTotal,
		Name:          sys.Name,
		ID:            sys.ID,
		KernelVersion: sys.KernelVersion,
		OSType:        sys.OSType,
		OS:            sys.OperatingSystem,
		DockerVersion: sys.ServerVersion,
	}

	return systemInfo, nil
}
