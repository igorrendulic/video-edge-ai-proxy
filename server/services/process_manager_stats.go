package services

import (
	"strings"

	"github.com/chryscloud/go-microkit-plugins/docker"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
)

// StatsAllProcesses created a statistics object or all running containers (related to edge)
func (pm *ProcessManager) StatsAllProcesses(sett *models.Settings) (*models.AllStreamProcessStats, error) {
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))

	systemInfo, diskUsage, err := cl.SystemWideInfo()

	stats := &models.AllStreamProcessStats{}
	// calculate disk usage and gather system info
	totalContainers := systemInfo.Containers
	runningContainers := systemInfo.ContainersRunning
	stoppedContainers := systemInfo.ContainersStopped
	totalImgSize := int64(0)
	activeImages := 0
	totalVolumeSize := int64(0)
	activeVolumes := int64(0)

	for _, im := range diskUsage.Images {
		activeImages += int(im.Containers)
		totalImgSize += im.SharedSize
	}
	for _, v := range diskUsage.Volumes {
		activeVolumes += v.UsageData.RefCount
		totalVolumeSize += v.UsageData.Size
	}

	stats.Containers = totalContainers
	stats.ContainersRunning = runningContainers
	stats.ContainersStopped = stoppedContainers
	stats.ActiveImages = int(activeImages)
	stats.TotalVolumeSize = totalVolumeSize
	stats.TotalActiveVolumes = int(activeVolumes)
	stats.GatewayID = sett.GatewayID
	stats.TotalImageSize = totalImgSize

	stats.ContainersStats = make([]*models.ProcessStats, 0)

	pList, err := pm.List()
	if err != nil {
		g.Log.Error("failed to list all containers", err)
		return nil, err
	}

	// gather all container stats
	for _, process := range pList {
		c, err := cl.ContainerGet(process.ContainerID)
		if err != nil {
			g.Log.Error("failed to get container from docker system", err)
			continue
		}
		n := c.Name
		// skip default running components
		if strings.Contains(n, "chrysedgeportal") || strings.Contains(n, "chrysedgeserver") || strings.Contains(n, "redis") {
			continue
		}

		s, err := cl.ContainerStats(c.ID)
		if err != nil {
			return nil, err
		}
		calculated := cl.CalculateStats(s)
		calculated.Status = c.State.Status
		restartCount := 0
		if c.State.ExitCode > 0 {
			restartCount = c.RestartCount
		}

		procStats := &models.ProcessStats{
			Name:        process.Name,
			ImageTag:    process.ImageTag,
			Cpu:         int(calculated.CPUPercent),
			Memory:      int(calculated.MemoryPercent),
			NetworkRx:   int64(calculated.NetworkRx),
			NetworkTx:   int64(calculated.NetworkTx),
			NumRestarts: restartCount,
			Status:      c.State.Status,
		}
		stats.ContainersStats = append(stats.ContainersStats, procStats)
	}

	return stats, nil
}
