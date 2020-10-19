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
	"github.com/chryscloud/microkit-plugins/dockerhub"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/dgraph-io/badger/v2"
	"github.com/hashicorp/go-version"
)

// SettingsManager - various settings for the edge
type SettingsManager struct {
	storage             *Storage
	current_edge_key    string
	current_edge_secret string
	mux                 *sync.RWMutex
}

func NewSettingsManager(storage *Storage) *SettingsManager {
	return &SettingsManager{
		storage: storage,
		mux:     &sync.RWMutex{},
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
func (sm *SettingsManager) Overwrite(new *models.Settings) error {
	settings, err := sm.getDefault()
	if err != nil {
		g.Log.Error("failed to retrieve default settings", err)
		return err
	}
	// curently only edgekey setting
	settings.EdgeKey = new.EdgeKey
	settings.EdgeSecret = new.EdgeSecret
	if settings.Created < 0 {
		settings.Created = time.Now().Unix() * 1000
	}
	settings.Modified = time.Now().Unix() * 1000

	settingsBytes, err := json.Marshal(settings)
	if err != nil {
		g.Log.Error("failed to marshal settings", err)
		return err
	}
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.current_edge_key = settings.EdgeKey
	sm.current_edge_secret = settings.EdgeSecret
	return sm.storage.Put(models.PrefixSettingsKey, settings.Name, settingsBytes)
}

func (sm *SettingsManager) Get() (*models.Settings, error) {
	return sm.getDefault()
}

func (sm *SettingsManager) ListDockerImages(nameTag string) (*models.ImageUpgrade, error) {
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))
	images, err := cl.ImagesList()
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

	resp := &models.ImageUpgrade{
		HasImage:             len(localTags) > 0,
		HasUpgrade:           hasUpgrade,
		Name:                 nameTag,
		HighestRemoteVersion: highestRemoteTagVersion,
		CurrentVersion:       highestLocalTagVersion,
	}

	return resp, nil
}

func (sm *SettingsManager) PullDockerImage(name, version string) (*models.PullDockerResponse, error) {
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))
	resp, err := cl.ImagePullDockerHub(name, version, "", "")
	if err != nil {
		g.Log.Error("failed to pull image from dockerhub", name, version, err)
		return nil, err
	}

	fmt.Printf("%v\n", resp)
	g.Log.Info(resp)

	response := &models.PullDockerResponse{
		Response: resp,
	}

	return response, nil
}

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
