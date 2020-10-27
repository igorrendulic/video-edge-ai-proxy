package services

import (
	"strings"

	"github.com/chryscloud/go-microkit-plugins/docker"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-version"
)

func (pm *ProcessManager) FindUpgrades(imageUpgrade *models.ImageUpgrade) ([]*models.StreamProcess, error) {
	processes, err := pm.List()

	if err != nil {
		g.Log.Error("failed to list local processes", err)
		return nil, err
	}

	currentVersion, vErr := version.NewVersion(imageUpgrade.CurrentVersion)
	if vErr != nil {
		g.Log.Error("version conversion failed", imageUpgrade.CurrentVersion, vErr)
		return nil, vErr
	}

	upgradesAvailable := make([]*models.StreamProcess, 0)

	for _, proc := range processes {
		imgTag := proc.ImageTag
		splitted := strings.Split(imgTag, ":")
		if len(splitted) == 2 {
			ver := splitted[1]
			processVersion, pErr := version.NewVersion(ver)
			if pErr != nil {
				g.Log.Warn("failed to convert version for", ver, pErr)
				continue
			}
			// check if upgrade available
			if currentVersion.GreaterThan(processVersion) {
				proc.UpgradeAvailable = true
				proc.NewerVersion = currentVersion.Original()
				upgradesAvailable = append(upgradesAvailable, proc)
			} else {
				upgradesAvailable = append(upgradesAvailable, proc)
			}
		} else {
			upgradesAvailable = append(upgradesAvailable, proc)
		}
	}

	return upgradesAvailable, nil
}

func (pm *ProcessManager) UpgradeRunningContainer(process *models.StreamProcess) error {
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))

	// find container
	containers, err := cl.ContainersList()
	if err != nil {
		g.Log.Error("failed to list running containers", err)
		return err
	}

	var runningContainer types.Container

	for _, container := range containers {
		names := container.Names
		if len(names) > 0 {
			name := names[0][1:]
			if name == process.Name {
				// found it
				runningContainer = container
			}
		}
	}

	if runningContainer.ID == "" {
		g.Log.Warn("container for process not found", process.Name, process.ImageTag, process.ContainerID)
		return ErrProcessNotFound
	}

	return nil
}
