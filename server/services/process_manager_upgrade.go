package services

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/chryscloud/go-microkit-plugins/docker"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/dgraph-io/badger/v2"
	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-version"
)

func (pm *ProcessManager) FindUpgrades(imageUpgrade *models.ImageUpgrade) ([]*models.StreamProcess, error) {
	processes, err := pm.List()

	if err != nil {
		g.Log.Error("failed to list local processes", err)
		return nil, err
	}

	upgradesAvailable := make([]*models.StreamProcess, 0)

	if imageUpgrade.CurrentVersion == "" {
		return upgradesAvailable, nil
	}

	currentVersion, vErr := version.NewVersion(imageUpgrade.CurrentVersion)
	if vErr != nil {
		g.Log.Error("version conversion failed", imageUpgrade.CurrentVersion, vErr)
		return nil, vErr
	}

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

func (pm *ProcessManager) UpgradeRunningContainer(process *models.StreamProcess, newImage string) (*models.StreamProcess, error) {
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))

	// find container
	containers, err := cl.ContainersList()
	if err != nil {
		g.Log.Error("failed to list running containers", err)
		return nil, err
	}

	var runningContainer types.Container

	for _, container := range containers {
		names := container.Names
		if len(names) > 0 {
			name := names[0][1:]
			if name == process.Name {
				// found it
				runningContainer = container
				break
			}
		}
	}

	if runningContainer.ID == "" {
		g.Log.Warn("container for process not found", process.Name, process.ImageTag, process.ContainerID)
		return nil, models.ErrProcessNotFound
	}

	// validate that the new version of image exists on disk
	existingImages, err := cl.ImagesList()
	if err != nil {
		g.Log.Error("failed to list local images", err)
		return nil, err
	}
	repoTagFound := false
	for _, img := range existingImages {
		repoTags := img.RepoTags
		for _, repoTag := range repoTags {
			if repoTag == newImage {
				// found it
				repoTagFound = true
			}
		}
	}
	if !repoTagFound {
		g.Log.Error("upgrade failed due to new image version doesn't exist (docker images)", newImage)
		return nil, errors.New("new image version donesn't exist")
	}

	// check if process exists in database
	processBytes, err := pm.storage.Get(models.PrefixRTSPProcess, process.Name)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, models.ErrProcessNotFound
		}
		g.Log.Error("failed to retrieve process from datastore", err)
		return nil, err
	}
	var existingProcess models.StreamProcess
	err = json.Unmarshal(processBytes, &existingProcess)
	if err != nil {
		g.Log.Error("failed to unmarshal existing process", err)
		return nil, err
	}

	splitted := strings.Split(newImage, ":")
	newVersion := splitted[1]
	newBaseImage := splitted[0]
	rErr := cl.ContainerReplace(runningContainer.ID, newBaseImage, newVersion)
	if rErr != nil {
		g.Log.Error("failed to replace running container", runningContainer.Names, runningContainer.ID, rErr)
		return nil, rErr
	}

	existingProcess.ImageTag = newBaseImage + ":" + newVersion
	existingProcess.Modified = time.Now().Unix() * 1000

	newExistingProcessBytes, err := json.Marshal(existingProcess)
	if err != nil {
		g.Log.Error("failed to marshal back to bytes existing process", err)
		return nil, err
	}

	err = pm.storage.Put(models.PrefixRTSPProcess, existingProcess.Name, newExistingProcessBytes)
	if err != nil {
		g.Log.Error("failed to store upgraded process", err)
		return nil, err
	}

	return &existingProcess, nil
}
