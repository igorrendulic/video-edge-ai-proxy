package models

// image upgrades object (when docker image doesn't exist or has an available update)
type ImageUpgrade struct {
	HasUpgrade           bool   `json:"has_upgrade"`
	HasImage             bool   `json:"has_image"`
	Name                 string `json:"name"`
	CurrentVersion       string `json:"current_version,omitempty"`
	HighestRemoteVersion string `json:"highest_remote_version"`
}

// Struct representing events returned from image pulling
type PullDockerResponse struct {
	Response string `json:"response,omitempty"`
}
