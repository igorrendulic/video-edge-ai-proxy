package mqtt

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
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
	err = mqtt.processService.Stop(payload.Name)
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
