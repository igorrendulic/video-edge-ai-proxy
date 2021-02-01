package models

// maximum size must not exceed 256KB, the smaller the better
type AllStreamProcessStats struct {
	GatewayID          string          `json:"gw"`  // gatewayID
	Containers         int             `json:"c"`   // total containers
	ContainersRunning  int             `json:"cr"`  // total containers currently running
	ContainersStopped  int             `json:"cs"`  // number of stopped containers in the system
	TotalImageSize     int64           `json:"is"`  // total image size
	ActiveImages       int             `json:"ia"`  // number of active images (used by containers)
	TotalActiveVolumes int             `json:"va"`  // total active volumes
	TotalVolumeSize    int64           `json:"vs"`  // total volume size
	ContainersStats    []*ProcessStats `json:"sts"` // general container stats
}

type ProcessStats struct {
	ImageTag    string `json:"it"` // image tag
	Name        string `json:"n"`  // name of the container
	Cpu         int    `json:"cp"` // cpu usage (in percent)
	NumRestarts int    `json:"nr"` // number of restarts
	Memory      int    `json:"m"`  // memory consumption
	NetworkRx   int64  `json:"x"`  // network read
	NetworkTx   int64  `json:"t"`  // network write
	Status      string `json:"s"`  // status
}

type SystemInfo struct {
	GatewayID     string `json:"gatewayId"`               // gatewayId
	RegistryID    string `json:"registryId"`              // registry id
	NCPUs         int    `json:"ncpu"`                    // number of CPUs
	Architecture  string `json:"architecture,omitempty"`  // e.g. x_86_64
	TotalMemory   int64  `json:"totalMemory,omitempty"`   // total memory
	Name          string `json:"name,omitempty"`          // name of the computer
	ID            string `json:"id,omitempty"`            // id of the computer
	KernelVersion string `json:"kernelVersion,omitempty"` // kernel version
	OSType        string `json:"osType,omitempty"`        // os type
	OS            string `json:"os,omitempty"`            // operating system
	DockerVersion string `json:"dockerVersion,omitempty"` // docker version
}
