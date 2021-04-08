package mqtt

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
	badger "github.com/dgraph-io/badger/v2"
	"github.com/docker/docker/api/types/events"
)

const (
	ProcessActionDie   = "die"
	ProcessActionStart = "start"
)

type ProcessState struct {
	Time     int64  // seconds since epoch
	DeviceID string // deviceID
	Action   string // process action from docker events
}

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

// detecting device state change and reporting if changes occured
func (mqtt *mqttManager) changedDeviceState(gatewayID string, message events.Message) error {

	actor := message.Actor

	// fairly complicated logic to handle container restarts and report only true changes, not attempty of restarting the container
	if deviceID, ok := actor.Attributes["name"]; ok {
		mqtt.mutex.Lock()
		defer mqtt.mutex.Unlock()

		var history []events.Message
		if val, ok := mqtt.processEvents.Load(deviceID); ok {
			history = val.([]events.Message)
			if len(history) >= 10 {
				startIndex := len(history) - 10
				history = history[startIndex:]
			}
			history = append(history, message)
		} else {
			history = []events.Message{message}
		}
		mqtt.processEvents.Store(deviceID, history)

		// check last value after 5 seconds (avoiding the possible burst of events for a specific container)
		go func(deviceID string) {
			time.Sleep(time.Second * 5)
			if val, ok := mqtt.processEvents.Load(deviceID); ok {
				history := val.([]events.Message)
				last := history[len(history)-1]
				// for _, last := range history {
				if lastNotified, ok := mqtt.lastProcessEventNotified.Load(deviceID); ok {
					if mqtt.hasDeviceDifferences(lastNotified.(events.Message), last) {
						stat := mqtt.deviceActionToStatus(last.Action)
						rErr := mqtt.reportDeviceStateChange(deviceID, stat)
						if rErr != nil {
							g.Log.Error("failed to report device state change", rErr)
							return
						}
						g.Log.Info("device status reported ", stat, deviceID)
					}
				} else {
					mqtt.lastProcessEventNotified.Store(deviceID, last)
					stat := mqtt.deviceActionToStatus(last.Action)
					rErr := mqtt.reportDeviceStateChange(deviceID, stat)
					if rErr != nil {
						g.Log.Error("failed to report device state change", rErr)
						return
					}
					g.Log.Info("device with no history yet; status reported ", stat, deviceID)
				}
			}
		}(deviceID)

	}

	return nil
}

// converting docker event name to status
func (mqtt *mqttManager) deviceActionToStatus(lastAction string) string {
	stat := models.ProcessStatusRestarting

	switch action := lastAction; action {
	case ProcessActionDie:
		stat = models.ProcessStatusRestarting
	case ProcessActionStart:
		stat = models.ProcessStatusRunning
	default:
		stat = lastAction
	}
	return stat
}

func (mqtt *mqttManager) reportDeviceStateChange(deviceID string, status string) error {

	tp := models.MQTTProcessType(models.ProcessTypeUnknown)

	var imageTag string
	var rtmpEndpoint string
	var rtspEndpoint string

	device, err := mqtt.processService.Info(deviceID)
	if err != nil {
		// check if application (prevent reporting of events happening not related to chrysalis)
		if err == models.ErrProcessNotFoundDatastore || err == models.ErrProcessNotFound {
			proc, pErr := mqtt.appService.Info(deviceID)
			if pErr != nil {
				if pErr == models.ErrProcessNotFoundDatastore || pErr == models.ErrProcessNotFound {
					return nil
				}
				g.Log.Error("failed to find application for reporting state change", pErr)
				return pErr
			}
			tp = models.MQTTProcessType(models.ProcessTypeApplication)
			imageTag = utils.ImageTagPartToString(proc.DockerHubUser, proc.DockerhubRepository, proc.DockerHubVersion)
		} else {
			g.Log.Error("failed to retrieve device info for reporting state change", err)
			return err
		}
	} else {
		tp = models.MQTTProcessType(models.ProcessTypeRTSP)
		rtmpEndpoint = device.RTMPEndpoint
		rtspEndpoint = device.RTSPEndpoint
	}

	sett, err := mqtt.settingsService.Get()
	if err != nil {
		g.Log.Error("failed to retrieve settings", err)
		return err
	}

	mqttMsg := &models.MQTTMessage{
		DeviceID:         deviceID,
		ImageTag:         imageTag,
		RTMPEndpoint:     rtmpEndpoint,
		RTSPConnection:   rtspEndpoint,
		State:            status,
		Created:          time.Now().UTC().Unix() * 1000,
		ProcessOperation: models.MQTTProcessOperation(models.DeviceOperationState),
		ProcessType:      tp,
	}
	pErr := utils.PublishMonitoringTelemetry(sett.GatewayID, (*mqtt.client), mqttMsg)
	if pErr != nil {
		g.Log.Error("Failed to publish monitoring telemetry", pErr)
		return pErr
	}
	return nil
}

// GatewayState reporting gateway state to ChrysalisCloud (the way for entire gateway to check in)
func (mqtt *mqttManager) gatewayState(gatewayID string) error {

	// report state to coreiot (this can be removed also)
	gatewayStateTopic := fmt.Sprintf("/devices/%s/state", gatewayID)
	gatewayInitPayload := fmt.Sprintf("%d", time.Now().Unix())

	if token := (*mqtt.client).Publish(gatewayStateTopic, 1, false, gatewayInitPayload); token.Wait() && token.Error() != nil {
		g.Log.Error("failed to publish initial gateway payload", token.Error())
		return token.Error()
	}

	// report state to chrysalis cloud
	g.Log.Info("Gateway state reported", time.Now())
	mqttMsg := &models.MQTTMessage{
		Created:          time.Now().UTC().Unix() * 1000,
		ProcessOperation: models.MQTTProcessOperation(models.GatewayOperationCheckIn),
	}
	pErr := utils.PublishMonitoringTelemetry(gatewayID, (*mqtt.client), mqttMsg)
	if pErr != nil {
		g.Log.Error("Failed to publish monitoring telemetry", pErr)
		return pErr
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
func (mqtt *mqttManager) hasDeviceDifferences(stored events.Message, current events.Message) bool {
	h1 := extractDeviceSignature(stored)
	h2 := extractDeviceSignature(current)
	return h1 != h2
}

// creates StreamProcess signature for it's main fields
func extractDeviceSignature(processMsg events.Message) string {
	payload := processMsg.Status + processMsg.Actor.ID
	h := md5.New()
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
