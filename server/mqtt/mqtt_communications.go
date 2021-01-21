package mqtt

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
	badger "github.com/dgraph-io/badger/v2"
	qtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-redis/redis/v7"
)

const (
	mqttBrokerURL      = "tls://mqtt.googleapis.com:8883"
	protocolVersion    = 4  // corresponds to MQTT 3.1.1
	minimumBackoffTime = 1  // initial backoff time in seconds
	maximumBackoffTime = 32 // maximum backoff time in seconds
)

var (
	backoffTime   = minimumBackoffTime
	shouldBackoff = false
)

// ProcessManager - start, stop of docker containers
type mqttManager struct {
	rdb             *redis.Client
	settingsService *services.SettingsManager
	processService  *services.ProcessManager
	client          *qtt.Client
	stop            chan bool
	gatewayID       string
	jwt             string
}

func NewMqttManager(rdb *redis.Client, settingsService *services.SettingsManager, processService *services.ProcessManager) *mqttManager {
	return &mqttManager{
		rdb:             rdb,
		settingsService: settingsService,
		processService:  processService,
	}
}

func (mqtt *mqttManager) onConnect(client qtt.Client) {
	g.Log.Info("MQTT client connected", client.IsConnected())
	shouldBackoff = false
	backoffTime = minimumBackoffTime

	mqtt.client = &client
}

func (mqtt *mqttManager) onMessage(client qtt.Client, msg qtt.Message) {
	g.Log.Info("Topic:", msg.Topic())
	g.Log.Info("Message: ", msg.Payload())
}

func (mqtt *mqttManager) onConnectionLost(client qtt.Client, err error) {
	g.Log.Error("MQTT connection lost", err)
}

func (mqtt *mqttManager) StartGatewayListener() error {
	err := mqtt.gatewayInit()

	if err != nil {
		g.Log.Error("failed to connect gateway and report presence", err)
		return err
	}

	// init redis listener for local messages (this is only for active local changes)
	// e.g. Device/process added, removed, ...
	sub := mqtt.rdb.Subscribe(models.RedisLocalMQTTChannel)

	go func() {

		defer sub.Close()

		val, err := sub.ReceiveMessage()
		if err != nil {
			g.Log.Error("failed to receive mqtt local pubsub message", err)
		} else {
			g.Log.Info("redis message received: ", val)
			payload := []byte(val.Payload)
			var localMsg models.MQTTMessage
			err := json.Unmarshal(payload, &localMsg)
			if err != nil {
				g.Log.Error("failed to unmarshal internal redis pubsub message", err)
			} else {
				g.Log.Info("Received message object from redis pubsub for mqtt: ", localMsg)
				var opErr error
				if localMsg.ProcessOperation == models.MQTTProcessOperation(models.DeviceOperationStart) {
					opErr = mqtt.bindDevice(localMsg.DeviceID)
				} else if localMsg.ProcessOperation == models.MQTTProcessOperation(models.DeviceOperationStop) {
					opErr = mqtt.unbindDevice(localMsg.DeviceID)
				} else if localMsg.ProcessOperation == models.MQTTProcessOperation(models.DeviceOperationUpgrade) {
					// TODO: TBD
				} else {
					opErr = errors.New("local message operation not recognized")
					g.Log.Error("message operation not recognized: ", localMsg.ProcessOperation, localMsg.DeviceID, localMsg.ProcessType)
				}
				if opErr != nil {
					g.Log.Error("local pubsub gateway msg failed", opErr)
				}
			}
		}
	}()

	// report gateway state every 60 seconds
	delay := time.Second * 60
	go func() {
		for {
			err := mqtt.gatewayState(mqtt.gatewayID)
			if err != nil {
				g.Log.Error("failed to report gateway state: ", err)
			}
			select {
			case <-time.After(delay):
			case <-mqtt.stop:
				return
			}
		}
	}()

	return nil
}

// GatewayState reporting gateway state to ChrysalisCloud
func (mqtt *mqttManager) gatewayState(gatewayID string) error {

	gatewayStateTopic := fmt.Sprintf("/devices/%s/state", gatewayID)
	gatewayInitPayload := fmt.Sprintf("start:%d", time.Now().Unix())

	if token := (*mqtt.client).Publish(gatewayStateTopic, 1, false, gatewayInitPayload); token.Wait() && token.Error() != nil {
		g.Log.Info("failed to publish initial gateway payload", token.Error())
		return token.Error()
	}
	g.Log.Info("Gateway state reported")
	return nil
}

// bind single device to this gateway
func (mqtt *mqttManager) bindDevice(deviceID string) error {
	deviceInfo, err := mqtt.processService.Info(deviceID)
	if err != nil {
		g.Log.Error("failed to query device info", err)
		return err
	}
	mqttMsg := &models.MQTTMessage{
		DeviceID:         deviceInfo.Name,
		Created:          time.Now().UTC().Unix() * 1000,
		ImageTag:         deviceInfo.ImageTag,
		ProcessOperation: models.MQTTProcessOperation(models.DeviceOperationStart),
		ProcessType:      models.MQTTProcessType(models.ProcessTypeDevice),
	}
	attErr := utils.AttachDeviceToGateway(mqtt.gatewayID, (*mqtt.client), mqttMsg)
	if attErr != nil {
		g.Log.Error("failed to attach ", deviceID, "to this gateway", attErr)
	}
	return nil
}

// unbiding single device from this gateway
func (mqtt *mqttManager) unbindDevice(deviceID string) error {
	deviceInfo, err := mqtt.processService.Info(deviceID)
	if err != nil {
		g.Log.Error("failed to query device info", err)
		return err
	}
	mqttMsg := &models.MQTTMessage{
		DeviceID:         deviceInfo.Name,
		Created:          time.Now().UTC().Unix() * 1000,
		ImageTag:         deviceInfo.ImageTag,
		ProcessOperation: models.MQTTProcessOperation(models.DeviceOperationStop),
		ProcessType:      models.MQTTProcessType(models.ProcessTypeDevice),
	}
	fmt.Printf("device info: %v %v\n", deviceInfo, mqttMsg)
	attErr := utils.DetachGatewayDevice(deviceInfo.Name, (*mqtt.client), mqttMsg)
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
		mqttMsg := &models.MQTTMessage{
			DeviceID:         device.Name,
			ImageTag:         device.ImageTag,
			Created:          device.Created,
			State:            device.State.Status,
			ProcessOperation: models.MQTTProcessOperation(models.DeviceOperationStart),
			ProcessType:      models.MQTTProcessType(models.ProcessTypeDevice),
		}
		attErr := utils.AttachDeviceToGateway(mqtt.gatewayID, (*mqtt.client), mqttMsg)
		if attErr != nil {
			g.Log.Error("failed to attach device", device.Name, attErr)
			hasErr = attErr
		}
	}
	return hasErr
}

// Start the MQTT communication gateway
func (mqtt *mqttManager) gatewayInit() error {

	// check settings if they exist
	settings, err := mqtt.settingsService.Get()
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil
		}
		g.Log.Error("failed to retrieve edge settings", err)
		return err
	}
	if settings.ProjectID == "" || settings.Region == "" || settings.GatewayID == "" || settings.RegistryID == "" || settings.PrivateRSAKey == nil {
		g.Log.Warn("ProjectID: ", settings.ProjectID, "Region: ", settings.Region, "GatewayID: ", settings.GatewayID, "RegistryID: ", settings.RegistryID)
		return nil
	}

	// rotate it every day at least (JWT token must expire sooner)
	jwt, ccErr := utils.CreateJWT(settings.ProjectID, settings.PrivateRSAKey, time.Hour*12)
	if ccErr != nil {
		g.Log.Error("Failed to create JWT key for communication with ChrysCloud MQTT", ccErr)
		return ccErr
	}
	clientID := fmt.Sprintf("projects/%s/locations/%s/registries/%s/devices/%s", settings.ProjectID, settings.Region, settings.RegistryID, settings.GatewayID)
	opts := qtt.NewClientOptions()
	opts.AddBroker(mqttBrokerURL)
	opts.SetClientID(clientID)
	opts.SetUsername("unused")
	opts.SetPassword(jwt)
	opts.SetProtocolVersion(protocolVersion)
	opts.SetOnConnectHandler(mqtt.onConnect)
	opts.SetDefaultPublishHandler(mqtt.onMessage)
	opts.SetConnectionLostHandler(mqtt.onConnectionLost)

	// Create and connect a client using the above options.
	client := qtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		g.Log.Error("failed to connect with mqtt ChrysCloud broker", token.Error())
		return token.Error()
	}

	mqtt.gatewayID = settings.GatewayID
	mqtt.jwt = jwt
	mqtt.client = &client

	err = mqtt.gatewayState(mqtt.gatewayID)
	if err != nil {
		g.Log.Error("Failed to send initial gateway state to chrysalis cloud", err)
		return err
	}

	errBind := mqtt.bindAllDevices()
	if errBind != nil {
		g.Log.Error("failed to report bind devices", errBind)
		return errBind
	}

	return nil
}

func (mqtt *mqttManager) StopGateway() error {
	if mqtt.client != nil {
		(*mqtt.client).Disconnect(20)
	}
	mqtt.stop <- true
	return nil
}
