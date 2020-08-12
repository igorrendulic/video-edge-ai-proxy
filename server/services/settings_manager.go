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
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/dgraph-io/badger/v2"
)

// SettingsManager - various settings for the edge
type SettingsManager struct {
	storage *Storage
}

func NewSettingsManager(storage *Storage) *SettingsManager {
	return &SettingsManager{
		storage: storage,
	}
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
	if settings.Created < 0 {
		settings.Created = time.Now().Unix() * 1000
	}
	settings.Modified = time.Now().Unix() * 1000

	settingsBytes, err := json.Marshal(settings)
	if err != nil {
		g.Log.Error("failed to marshal settings", err)
		return err
	}
	return sm.storage.Put(models.PrefixSettingsKey, settings.Name, settingsBytes)
}

func (sm *SettingsManager) Get() (*models.Settings, error) {
	return sm.getDefault()
}
