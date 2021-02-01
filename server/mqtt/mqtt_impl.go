package mqtt

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
	badger "github.com/dgraph-io/badger/v2"
)

// Check settings and also if MQTT initial connection has been made
func (mqtt *mqttManager) getMQTTSettings() (*models.Settings, error) {
	// check settings if they exist
	settings, err := mqtt.settingsService.Get()
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, ErrNoMQTTSettings
		}
		g.Log.Error("failed to retrieve edge settings", err)
		return nil, err
	}
	if settings.ProjectID == "" || settings.Region == "" || settings.GatewayID == "" || settings.RegistryID == "" || settings.PrivateRSAKey == nil {
		return nil, ErrNoMQTTSettings
	}
	return settings, nil
}

// config and commans subscribers
func (mqtt *mqttManager) gatewaySubscribers() error {
	// wait for connection to be opened and propagate

	errBind := mqtt.bindAllDevices()
	if errBind != nil {
		g.Log.Error("failed to report bind devices", errBind)
		return errBind
	}

	errCfg := mqtt.subscribeToConfig(mqtt.gatewayID)
	if errCfg != nil {
		g.Log.Error("failed to subscribe to mqtt config subscription", mqtt.gatewayID, errCfg)
		return errCfg
	}

	errCmd := mqtt.subscribeToCommands(mqtt.gatewayID)
	if errCmd != nil {
		g.Log.Error("failed to subscribe to mqtt commands", mqtt.gatewayID, errCmd)
		return errCmd
	}

	return nil
}

// GatewayState reporting gateway state to ChrysalisCloud
func (mqtt *mqttManager) gatewayState(gatewayID string) error {

	gatewayStateTopic := fmt.Sprintf("/devices/%s/state", gatewayID)
	gatewayInitPayload := fmt.Sprintf("%d", time.Now().Unix())

	if token := (*mqtt.client).Publish(gatewayStateTopic, 1, false, gatewayInitPayload); token.Wait() && token.Error() != nil {
		g.Log.Error("failed to publish initial gateway payload", token.Error())
		return token.Error()
	}

	g.Log.Info("Gateway state reported", time.Now())

	// report changes of all the stream processes to events (find diff)
	allDevices, err := mqtt.processService.List()
	if err != nil {
		g.Log.Error("failed to list all devices", err)
		return err
	}
	sett, err := mqtt.settingsService.Get()
	if err != nil {
		g.Log.Error("failed to retrieve settings", err)
		return err
	}
	if sett.GatewayID == "" {
		g.Log.Error("gatewayID not exists in settings")
		return errors.New("settings do not have gateway id")
	}

	// check if anything changed from the last report and push changes to chrysalis cloud if it did
	for _, device := range allDevices {
		report := false
		if dev, ok := mqtt.latestDeviceState[device.Name]; ok {
			// simple check for differences
			report = mqtt.hasDeviceDifferences(dev, device)
		} else {
			// if not processed then send it's status
			report = true
		}
		if report {
			tp := models.MQTTProcessType(models.ProcessTypeUnknown)
			if device.RTSPEndpoint != "" {
				tp = models.MQTTProcessType(models.ProcessTypeRTSP)
			}
			mqttMsg := &models.MQTTMessage{
				DeviceID:         device.Name,
				ImageTag:         device.ImageTag,
				RTMPEndpoint:     device.RTMPEndpoint,
				RTSPConnection:   device.RTSPEndpoint,
				State:            device.State.Status,
				Created:          time.Now().UTC().Unix() * 1000,
				ProcessOperation: models.MQTTProcessOperation(models.DeviceOperationState),
				ProcessType:      tp,
			}
			pErr := utils.PublishMonitoringTelemetry(sett.GatewayID, (*mqtt.client), mqttMsg)
			if pErr != nil {
				g.Log.Error("Failed to publish monitoring telemetry", pErr)
				return pErr
			}
			mqtt.latestDeviceState[device.Name] = device
		}
	}

	return nil
}

// subscribing to mqtt config notifications from ChrysalisCloud
func (mqtt *mqttManager) subscribeToConfig(gatewayID string) error {
	config := fmt.Sprintf("/devices/%s/config", gatewayID)
	if token := (*mqtt.client).Subscribe(config, 1, mqtt.configHandler); token.Wait() && token.Error() != nil { // using default handler
		g.Log.Error("failed to subscribe to ", config, token.Error())
		return token.Error()
	}
	g.Log.Info("Subscribed to mqtt config topic")
	return nil
}

// subscribing to mqtt commands
func (mqtt *mqttManager) subscribeToCommands(gatewayID string) error {
	comm := fmt.Sprintf("/devices/%s/commands/#", gatewayID)
	if token := (*mqtt.client).Subscribe(comm, 1, nil); token.Wait() && token.Error() != nil {
		g.Log.Error("Failed to subscribe to mqtt commands", comm, token.Error())
		return token.Error()
	}
	return nil
}

// bind single device to this gateway
func (mqtt *mqttManager) bindDevice(deviceID string, processType models.MQTTProcessType) error {
	device, err := mqtt.processService.Info(deviceID)
	if err != nil {
		return err
	}
	mqttMsg := &models.MQTTMessage{
		DeviceID:         device.Name,
		ImageTag:         device.ImageTag,
		RTMPEndpoint:     device.RTMPEndpoint,
		RTSPConnection:   device.RTSPEndpoint,
		State:            device.State.Status,
		Created:          time.Now().UTC().Unix() * 1000,
		ProcessOperation: models.MQTTProcessOperation(models.DeviceOperationAdd),
		ProcessType:      processType,
	}
	attErr := utils.AttachDeviceToGateway(mqtt.gatewayID, (*mqtt.client), mqttMsg)
	if attErr != nil {
		g.Log.Error("failed to attach ", device.Name, "to this gateway", attErr)
	}
	return nil
}

// unbiding single device from this gateway
func (mqtt *mqttManager) unbindDevice(deviceID string, processType models.MQTTProcessType) error {
	set, err := mqtt.settingsService.Get()
	if err != nil {
		return err
	}
	mqttMsg := &models.MQTTMessage{
		DeviceID:         deviceID,
		Created:          time.Now().UTC().Unix() * 1000,
		ProcessOperation: models.MQTTProcessOperation(models.DeviceOperationRemove),
		ProcessType:      processType,
	}

	attErr := utils.DetachGatewayDevice(set.GatewayID, (*mqtt.client), mqttMsg)
	if attErr != nil {
		g.Log.Error("failed to dettach ", deviceID, "from this gateway", attErr)
	}
	return nil
}

// Lists all processes (running ones) and binds them to this gateway
func (mqtt *mqttManager) bindAllDevices() error {
	all, err := mqtt.processService.List()
	if err != nil {
		g.Log.Error("failed to list all processes", err)
		return err
	}
	var hasErr error
	for _, device := range all {
		processType := models.ProcessTypeUnknown
		if strings.Contains(device.ImageTag, "chrysedgeproxy") {
			processType = models.ProcessTypeRTSP
		}
		mqttMsg := &models.MQTTMessage{
			DeviceID:         device.Name,
			ImageTag:         device.ImageTag,
			Created:          device.Created,
			State:            device.State.Status,
			ProcessOperation: models.MQTTProcessOperation(models.DeviceOperationAdd),
			ProcessType:      models.MQTTProcessType(processType),
		}
		attErr := utils.AttachDeviceToGateway(mqtt.gatewayID, (*mqtt.client), mqttMsg)
		if attErr != nil {
			g.Log.Error("failed to attach device", device.Name, attErr)
			hasErr = attErr
		}
	}
	return hasErr
}

// hasDeviceDifferences checks if the previously reported device has changed (only important fields check):
// status, state, rtsp link, rtmp link, containerID
func (mqtt *mqttManager) hasDeviceDifferences(storedDeviceState *models.StreamProcess, current *models.StreamProcess) bool {
	h1 := extractDeviceSignature(storedDeviceState)
	h2 := extractDeviceSignature(current)
	return h1 != h2
}

// creates StreamProcess signature for it's main fields
func extractDeviceSignature(device *models.StreamProcess) string {
	payload := device.Status + device.ContainerID + fmt.Sprintf("%d", device.Created) + device.ImageTag + device.Name + device.RTMPEndpoint + device.RTSPEndpoint
	h := md5.New()
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
