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

const (
	PrefixSettingsKey               = "/settings/"
	PrefixSettingsDockerTagVersions = "/dockertagsettings/"

	SettingDefaultKey = "default"
)

// Settings - keeping setting on the edge
type Settings struct {
	Name          string `json:"name"`                      // name of the setting
	EdgeKey       string `json:"edge_key,omitempty"`        // edge key generated on Chrysalis Cloud
	EdgeSecret    string `json:"edge_secret,omitempty"`     // edge secret key generated on Chrysalis Cloud
	ProjectID     string `json:"project_id,omitempty"`      // Cloud project ID
	Region        string `json:"region,omitempty"`          // Edge region
	RegistryID    string `json:"registry_id,omitempty"`     // RegistryID
	GatewayID     string `json:"gateway_id,omitempty"`      // gatewayID
	PrivateRSAKey []byte `json:"private_rsa_key,omitempty"` // private RSA key for this gateway
	Created       int64  `json:"created,omitempty"`
	Modified      int64  `json:"modified,omitempty"`
}

// SettingsDockerTagVersion - current docker tag and version of the e.g. RTSP camera docker image
type SettingDockerTagVersion struct {
	Tag        string `json:"tag" binding:"required"`
	Version    string `json:"version" binding:"required"`
	CameraType string `json:"camera_type" binding:"required"` // e.g. rtsp for rtsp camera
}
