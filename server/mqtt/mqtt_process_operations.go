package mqtt

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/chryscloud/go-microkit-plugins/docker"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
)

// Removes a camera from the edge
func (mqtt *mqttManager) StopCamera(configPayload []byte) error {
	g.Log.Info("received payload to remove a device")
	var payload models.EdgeCommandPayload
	err := json.Unmarshal(configPayload, &payload)
	if err != nil {
		g.Log.Error("failed to unmarshal config payload", err)
		return err
	}
	_, pErr := mqtt.processService.Info(payload.Name)
	if pErr != nil {
		if pErr == models.ErrProcessNotFound {
			// nothing to do, but report unbinding
			err = mqtt.unbindDevice(payload.Name, models.MQTTProcessType(models.ProcessTypeRTSP))
			if err != nil {
				g.Log.Error("failed to publish binding event to chrysalis cloud of the new device", err)
				return err
			}
		}
		return pErr
	}

	// process found, can delete
	err = mqtt.processService.Stop(payload.Name, models.PrefixRTSPProcess)
	if err != nil {
		g.Log.Info("failed to delete process from edge", err)
		return err
	}

	// report unbiding of device to chrysalis cloud
	err = mqtt.unbindDevice(payload.Name, models.MQTTProcessType(models.ProcessTypeRTSP))
	if err != nil {
		g.Log.Error("failed to publish binding event to chrysalis cloud of the new device", err)
		return err
	}
	return nil
}

// Starts a new camera on the edge
func (mqtt *mqttManager) StartCamera(configPayload []byte) error {
	g.Log.Info("received payload to start a new camera")
	var payload models.EdgeCommandPayload
	err := json.Unmarshal(configPayload, &payload)
	if err != nil {
		g.Log.Error("failed to unmarshal config payload", err)
		return err
	}

	// check if camera already installed

	streamProcess := &models.StreamProcess{
		Name:         payload.Name,
		ImageTag:     payload.ImageTag,
		RTSPEndpoint: payload.RTSPEndpoint,
		RTMPEndpoint: payload.RTMPEndpoint,
		Created:      time.Now().Unix() * 1000,
		RTMPStreamStatus: &models.RTMPStreamStatus{
			Streaming: true,
			Storing:   false,
		},
	}
	_, pErr := mqtt.processService.Info(streamProcess.Name)
	if pErr == nil {
		// already running, nothing to do but report it's here
		err = mqtt.bindDevice(streamProcess.Name, models.MQTTProcessType(models.ProcessTypeRTSP))
		if err != nil {
			g.Log.Error("failed to publish binding event to chrysalis cloud of the new device", err)
			return err
		}
	}

	rtspImageTag := models.CameraTypeToImageTag[payload.Type]
	if rtspImageTag == "" {
		g.Log.Error("failed to find payload type", payload.Type)
		return errors.New("no payload type for " + payload.Type)
	}
	highestImgVersion, err := mqtt.settingsService.ListDockerImages(rtspImageTag)
	if err != nil {
		g.Log.Error("failed to list currently available images", err)
		return err
	}
	// if image doesn't exist, pull it down (this is in case where edge hasn't been initialized yet with specified docker image)
	if !highestImgVersion.HasImage {
		splitted := strings.Split(streamProcess.ImageTag, ":")

		_, err := mqtt.settingsService.PullDockerImage(splitted[0], splitted[1])
		if err != nil {
			g.Log.Error("failed to pull specified version", streamProcess.ImageTag, err)
			return err
		}
	}
	// re-list local docker images
	highestImgVersion, err = mqtt.settingsService.ListDockerImages(rtspImageTag)
	if err != nil {
		g.Log.Error("failed to list currently available images", err)
		return err
	}

	g.Log.Info(highestImgVersion)

	err = mqtt.processService.Start(streamProcess, highestImgVersion)
	if err != nil {
		g.Log.Error("failed to start new device", streamProcess.Name, streamProcess.ImageTag, streamProcess.RTSPEndpoint, err)
		return err
	}

	err = mqtt.bindDevice(streamProcess.Name, models.MQTTProcessType(models.ProcessTypeRTSP))
	if err != nil {
		g.Log.Error("failed to publish binding event to chrysalis cloud of the new device", err)
		return err
	}

	return nil
}

// Report to the cloud all container stats in one go
func (mqtt *mqttManager) ReportContainersStats() error {

	sett, err := mqtt.settingsService.Get()
	if err != nil {
		g.Log.Error("failed to get default settings", err)
		return err
	}
	if sett.GatewayID == "" || sett.RegistryID == "" {
		g.Log.Error("failed to report full system stats, gatewayID or registryID in settings missing", sett.GatewayID, sett.RegistryID)
		return errors.New("missing gateway or report id in settings")
	}
	procStats, err := mqtt.processService.StatsAllProcesses(sett)
	if err != nil {
		g.Log.Error("failed to retrieve all process stats", err)
		return err
	}

	statsBytes, err := json.Marshal(procStats)
	if err != nil {
		g.Log.Error("failed to marshalall process streams to report to cloud", err)
		return err
	}

	mqttMsg := &models.MQTTMessage{
		Created:          time.Now().UTC().Unix() * 1000,
		ProcessOperation: models.MQTTProcessOperation(models.DeviceOperationStats),
		ProcessType:      models.MQTTProcessType(models.ProcessTypeStats),
		Message:          statsBytes,
	}
	pErr := utils.PublishMonitoringTelemetry(sett.GatewayID, (*mqtt.client), mqttMsg)
	if pErr != nil {
		g.Log.Error("Failed to publish monitoring telemetry", pErr)
		return pErr
	}
	return nil
}

// PullApplication - pull/refresh docker image
func (mqtt *mqttManager) PullApplication(configPayload []byte) (*models.EdgeCommandPayload, error) {
	g.Log.Info("received payload to start installing new app")

	var payload models.EdgeCommandPayload
	err := json.Unmarshal(configPayload, &payload)
	if err != nil {
		g.Log.Error("failed to unmarshal app config payload", err)
		return nil, err
	}

	// check if app already pulled
	cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))

	images, err := cl.ImagesList()
	if err != nil {
		g.Log.Error("failed to retrieve container list", err)
		return nil, err
	}

	for _, im := range images {
		for _, tag := range im.RepoTags {
			if tag == payload.ImageTag {
				return &payload, nil
			}
		}
	}

	// notify cloud about pulling down the image
	mqtt.notifyMqtt(payload.Name, payload.ImageTag, models.MQTTProcessOperation(models.DeviceOperationAdd), models.MQTTProcessType(payload.Type), models.ProcessStatusInProgress, "Image pull")

	splitted := strings.Split(payload.ImageTag, ":")
	if len(splitted) == 2 {
		pullResponse, pullErr := mqtt.settingsService.PullDockerImage(splitted[0], splitted[1])
		if pullErr != nil {
			mqtt.notifyMqtt(payload.Name, payload.ImageTag, models.MQTTProcessOperation(models.DeviceOperationError), models.MQTTProcessType(payload.Type), models.ProcessStatusFailed, "Pull failed")
			g.Log.Error("failed to pull docker app", pullErr, pullResponse)
			return nil, pullErr
		}
	} else {
		// report error to cloud
		mqtt.notifyMqtt(payload.Name, payload.ImageTag, models.MQTTProcessOperation(models.DeviceOperationError), models.MQTTProcessType(payload.Type), models.ProcessStatusFailed, "ImageTag parse failed")
		return nil, errors.New("failed to parse imageTag: " + payload.ImageTag)
	}

	return &payload, nil
}

// StartApplication - start the application and report status to cloud (this method can assume image has been pulled succesfully)
func (mqtt *mqttManager) StartApplication(payload *models.EdgeCommandPayload) error {

	dockerUser, dockerRepo, dockerVersion := utils.ImageTagToParts(payload.ImageTag)

	app := &models.AppProcess{
		Name:                payload.Name,
		Runtime:             payload.Runtime,
		DockerHubUser:       dockerUser,
		DockerhubRepository: dockerRepo,
		DockerHubVersion:    dockerVersion,
	}

	var varArgs []*models.VarPair
	var envVars []*models.VarPair
	var mounts []*models.VarPair
	var portMapping []*models.PortMap
	if len(payload.ArgVars) > 0 {
		varArgs = utils.StringPairsToVarPairs(payload.ArgVars)
	}
	if len(payload.EnvVars) > 0 {
		envVars = utils.StringPairsToVarPairs(payload.EnvVars)
	}
	if len(payload.Mounts) > 0 {
		mounts = utils.StringPairsToVarPairs(payload.Mounts)
	}
	if len(payload.PortMapping) > 0 {
		portMaps := utils.StringPairsToVarPairs(payload.PortMapping)
		for _, pair := range portMaps {
			from, err := strconv.Atoi(pair.Name)
			to, err := strconv.Atoi(pair.Value)
			if err != nil {
				continue
			}
			pm := &models.PortMap{
				Exposed: from,
				MapTo:   to,
			}
			portMapping = append(portMapping, pm)
		}
	}
	app.EnvVars = envVars
	app.ArgsVars = varArgs
	app.MountFolders = mounts
	app.PortMapping = portMapping

	app, err := mqtt.appService.Install(app)
	if err != nil {
		if err == models.ErrProcessConflict {
			mqtt.notifyMqtt(payload.Name, payload.ImageTag, models.MQTTProcessOperation(models.DeviceOperationError), models.MQTTProcessType(payload.Type), models.ProcessStatusFailed, "name conflict")
			return err
		}
		g.Log.Error("failed to install application", payload.Name, err)
		mqtt.notifyMqtt(payload.Name, payload.ImageTag, models.MQTTProcessOperation(models.DeviceOperationError), models.MQTTProcessType(payload.Type), models.ProcessStatusFailed, "install failed")
		return err
	}

	mqtt.notifyMqtt(payload.Name, payload.ImageTag, models.MQTTProcessOperation(models.DeviceOperationAdd), models.MQTTProcessType(payload.Type), app.Status, "")
	return nil
}

// stop the application (doesn't remove docker image)
func (mqtt *mqttManager) StopApplication(configPayload []byte) error {
	var payload models.EdgeCommandPayload
	err := json.Unmarshal(configPayload, &payload)
	if err != nil {
		g.Log.Error("failed to unmarshal app config payload", err)
		mqtt.notifyMqtt(payload.Name, payload.ImageTag, models.MQTTProcessOperation(models.DeviceOperationError), models.MQTTProcessType(payload.Type), models.ProcessStatusFailed, "stop failed")
		return err
	}

	err = mqtt.processService.Stop(payload.Name, models.PrefixAppProcess)
	if err != nil {
		// only report error if process exists
		if err != models.ErrProcessNotFound {
			g.Log.Warn("failed to start process ", payload.Name, err)
			mqtt.notifyMqtt(payload.Name, payload.ImageTag, models.MQTTProcessOperation(models.DeviceOperationError), models.MQTTProcessType(payload.Type), models.ProcessStatusFailed, "stop failed")
			return err
		}
	}
	// publish to chrysalis cloud the change
	mqtt.notifyMqtt(payload.Name, payload.ImageTag, models.MQTTProcessOperation(models.DeviceOperationRemove), models.MQTTProcessType(payload.Type), models.ProcessStatusExited, "")
	return nil
}

// mqtt notification message to chrys cloud
func (mqtt *mqttManager) notifyMqtt(appName string, imageTag string, operation models.MQTTProcessOperation, operationType models.MQTTProcessType, status string, msg string) error {

	set, err := mqtt.settingsService.Get()
	if err != nil {
		g.Log.Error("failed to retrieve settings", err)
		return err
	}

	var message []byte

	if msg != "" {
		message = []byte(msg)
	}

	payload := &models.MQTTMessage{
		DeviceID:         appName,
		ImageTag:         imageTag,
		ProcessOperation: operation,
		ProcessType:      operationType,
		State:            status,
		Message:          message,
	}
	err = utils.PublishOperationTelemetry(set.GatewayID, (*mqtt.client), payload)
	if err != nil {
		g.Log.Error("failed to report operation status to chrys cloud", err)
		return err
	}
	return nil
}
