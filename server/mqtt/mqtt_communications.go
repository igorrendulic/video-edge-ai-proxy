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
	mqttBrokerURL   = "tls://mqtt.googleapis.com:8883"
	protocolVersion = 4 // corresponds to MQTT 3.1.1
)

// ProcessManager - start, stop of docker containers
type mqttManager struct {
	rdb               *redis.Client
	settingsService   *services.SettingsManager
	processService    *services.ProcessManager
	client            *qtt.Client
	clientOpts        *qtt.ClientOptions
	stop              chan bool
	gatewayID         string
	projectID         string
	jwt               string
	latestDeviceState map[string]*models.StreamProcess
}

func NewMqttManager(rdb *redis.Client, settingsService *services.SettingsManager, processService *services.ProcessManager) *mqttManager {
	return &mqttManager{
		rdb:               rdb,
		settingsService:   settingsService,
		processService:    processService,
		latestDeviceState: make(map[string]*models.StreamProcess),
	}
}

func (mqtt *mqttManager) onConnect(client qtt.Client) {
	g.Log.Info("MQTT client connected", client.IsConnected())
}

func (mqtt *mqttManager) onMessage(client qtt.Client, msg qtt.Message) {
	// TODO: commands should come in here
	g.Log.Info("Topic:", msg.Topic())
	g.Log.Info("Message: ", string(msg.Payload()))
}

func (mqtt *mqttManager) onConnectionLost(client qtt.Client, err error) {
	g.Log.Error("MQTT connection lost", err)
}

func (mqtt *mqttManager) configHandler(client qtt.Client, msg qtt.Message) {
	// TODO: update config here
	g.Log.Info("Topic:", msg.Topic())
	g.Log.Info("Message: ", string(msg.Payload()))
}

// StartGatewayListener checks every 15 seconds if there are any settings for connection to gateway
func (mqtt *mqttManager) StartGatewayListener() error {

	delay := time.Second * 15
	go func() {

		for {
			_, err := mqtt.getMQTTSettings()
			if err == nil {
				mqttErr := mqtt.run()
				if mqttErr != nil {
					g.Log.Error("Failed to init mqtt", mqttErr)
				}
				// exit the waiting function
				break
			}

			select {
			case <-time.After(delay):
			case <-mqtt.stop:
				g.Log.Info("MQTT cron job stopped")
				return
			}
		}
	}()

	return nil
}

func (mqtt *mqttManager) run() error {
	err := mqtt.gatewayInit()

	if err != nil {
		if err == ErrNoMQTTSettings {
			return nil
		}
		g.Log.Error("failed to connect gateway and report presence", err)
		return err
	}

	// init redis listener for local messages (this is only for active local changes)
	// e.g. Device/process added, removed, ...
	sub := mqtt.rdb.Subscribe(models.RedisLocalMQTTChannel)

	go func(sub *redis.PubSub) {

		defer sub.Close()

		for {
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
					if localMsg.ProcessOperation == models.MQTTProcessOperation(models.DeviceOperationAdd) {

						opErr = mqtt.bindDevice(localMsg.DeviceID, models.MQTTProcessType(models.ProcessTypeRTSP))

					} else if localMsg.ProcessOperation == models.MQTTProcessOperation(models.DeviceOperationRemove) {

						opErr = mqtt.unbindDevice(localMsg.DeviceID, models.MQTTProcessType(models.ProcessTypeRTSP))

					} else if localMsg.ProcessOperation == models.MQTTProcessOperation(models.DeviceOperationUpgradeAvailable) {
						// TODO: TBD
						g.Log.Warn("TBD: process operation upgrade available")
					} else if localMsg.ProcessOperation == models.MQTTProcessOperation(models.DeviceOperationUpgradeFinished) {
						// TODO: TBD
						g.Log.Warn("TBD: process operation upgrade completed/finished")
					} else {
						opErr = errors.New("local message operation not recognized")
						g.Log.Error("message operation not recognized: ", localMsg.ProcessOperation, localMsg.DeviceID, localMsg.ProcessType)
					}
					if opErr != nil {
						g.Log.Error("local pubsub gateway msg failed", opErr)
					}
				}
			}
		}
	}(sub)

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
				g.Log.Info("MQTT cron job stopped")
				return
			}
		}
	}()

	return nil
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
		return ErrNoMQTTSettings
	}

	// rotate it every day at least (JWT token must expire sooner)
	jwt, ccErr := utils.CreateJWT(settings.ProjectID, settings.PrivateRSAKey, time.Hour*1)
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
	opts.SetCleanSession(false)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(time.Second * 15)

	mqtt.gatewayID = settings.GatewayID
	mqtt.projectID = settings.ProjectID
	mqtt.jwt = jwt
	mqtt.clientOpts = opts

	cl, cErr := mqtt.connectClient(opts, settings, jwt)
	if cErr != nil {
		g.Log.Error("failed to connect client", cErr)
		return cErr
	}

	mqtt.client = cl

	mqtt.monitorTokenExpiration()

	return nil
}

func (mqtt *mqttManager) connectClient(opts *qtt.ClientOptions, settings *models.Settings, jwt string) (*qtt.Client, error) {
	// Create and connect a client using the above options.
	client := qtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		g.Log.Error("failed to connect with mqtt ChrysCloud broker", token.Error())
		return nil, token.Error()
	}

	mqtt.client = &client

	for {
		time.Sleep(time.Second * 5)

		// register subscribers
		err := mqtt.gatewaySubscribers()
		if err == nil {
			break
		}
		g.Log.Error("failed to initialize subscribers", err)
	}
	return &client, nil
}

func (mqtt *mqttManager) StopGateway() error {
	g.Log.Info("mqtt disconnect")
	if mqtt.client != nil {
		(*mqtt.client).Disconnect(20)
	}
	mqtt.stop <- true
	return nil
}

// monitoring the connection state every 15 seconds (also handles jwt expired tokens)
func (mqtt *mqttManager) monitorTokenExpiration() error {

	delay := time.Second * 15
	go func() {
		for {

			expirationTime, err := utils.ParseJWTTokenExpirationTime(mqtt.jwt)
			if err != nil {
				g.Log.Error("failed ot parse jwt tokens expiration time: ", err)
				return
			}
			today := time.Now().UTC().Unix() * 1000

			diff := today - (expirationTime.Unix() * 1000)

			if diff >= -(60 * 1000) {
				g.Log.Info("Re-issuing JWT token and re-connecting MQTT client", diff)
				sett, err := mqtt.settingsService.Get()
				if err != nil {
					g.Log.Error("failed to retrieve settings", sett)
					return
				}
				cl := (*mqtt.client)

				cl.Disconnect(300)

				jwt, ccErr := utils.CreateJWT(sett.ProjectID, sett.PrivateRSAKey, time.Hour*1)
				if ccErr != nil {
					g.Log.Error("Failed to create JWT key for communication with ChrysCloud MQTT", ccErr)
					return
				}
				mqtt.clientOpts.SetPassword(jwt)
				mqtt.jwt = jwt
				_, cErr := mqtt.connectClient(mqtt.clientOpts, sett, jwt)
				if cErr != nil {
					g.Log.Error("failed to reconnect client", cErr)
					return
				}
			}

			select {
			case <-time.After(delay):
			case <-mqtt.stop:
				g.Log.Info("mqtt stopped")
				return
			}
		}
	}()

	return nil
}