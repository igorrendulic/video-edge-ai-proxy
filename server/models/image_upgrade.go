package models

var (
	ImageTagVersionToCameraType = map[string]string{"chryscloud/chrysedgeproxy": "rtsp"}
	CameraTypeToImageTag        = map[string]string{"rtsp": "chryscloud/chrysedgeproxy"}
)

// image upgrades object (when docker image doesn't exist or has an available update)
type ImageUpgrade struct {
	HasUpgrade           bool   `json:"has_upgrade"`
	HasImage             bool   `json:"has_image"`
	Name                 string `json:"name"`
	CurrentVersion       string `json:"current_version,omitempty"`
	HighestRemoteVersion string `json:"highest_remote_version"`
	CameraType           string `json:"camera_type"`
}

// Struct representing events returned from image pulling
type PullDockerResponse struct {
	Response string `json:"response,omitempty"`
}
