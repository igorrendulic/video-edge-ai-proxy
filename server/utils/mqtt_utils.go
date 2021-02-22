package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/dgrijalva/jwt-go"
	qtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-redis/redis/v7"
)

// CreateJWT creates RS265 JWT signed token
func CreateJWT(projectID string, privateKeyBytes []byte, expiration time.Duration) (string, error) {
	claims := jwt.StandardClaims{
		Audience:  projectID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(expiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims)

	algorithm := "RS256"

	switch algorithm {
	case "RS256":
		privKey, pErr := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
		if pErr != nil {
			g.Log.Error("invalid private key", pErr)
			return "", pErr
		}
		return token.SignedString(privKey)
	case "ES256":
		privKey, _ := jwt.ParseECPrivateKeyFromPEM(privateKeyBytes)
		return token.SignedString(privKey)
	}

	return "", errors.New("Cannot find JWT algorithm. Specify 'ES256' or 'RS256'")
}

// ParseJWTTokenExpirationTime (no validation parsing of the jwt token in string format)
func ParseJWTTokenExpirationTime(jwtToken string) (time.Time, error) {
	claims := jwt.MapClaims{}
	token, _, err := new(jwt.Parser).ParseUnverified(jwtToken, jwt.MapClaims{})
	if err != nil {
		return time.Time{}, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return time.Time{}, errors.New("Can't convert token's claims to standard claims")
	}
	var tm time.Time
	switch exp := claims["exp"].(type) {
	case float64:
		tm = time.Unix(int64(exp), 0).UTC()
	case json.Number:
		v, _ := exp.Int64()
		tm = time.Unix(v, 0).UTC()
	}
	return tm, nil
}

// publishingTelemtry to gateway with custom quality of service
// 0 - at most one
// 1 - at least once
// 2 - exactly once
func publishTelemetry(gatewayID string, client qtt.Client, mqttMsg *models.MQTTMessage) error {
	telemetry := fmt.Sprintf("/devices/%v/events", gatewayID)

	mqttBytes, err := json.Marshal(mqttMsg)
	if err != nil {
		g.Log.Error("failed to marshal mqtt message", err)
		return err
	}

	if token := client.Publish(telemetry, 1, true, mqttBytes); token.WaitTimeout(time.Second*5) && token.Error() != nil {
		g.Log.Info("failed to publish initial gateway payload", token.Error())
		return token.Error()
	}
	return nil
}

// Publishing operation telemetry (applicaton and camera related operation such as install, remove, ...)
func PublishOperationTelemetry(gatewayID string, client qtt.Client, mqttMsg *models.MQTTMessage) error {
	return publishTelemetry(gatewayID, client, mqttMsg)
}

// Publish monitoring uses qos 0 (no biggy if we miss an event or two)
func PublishMonitoringTelemetry(gatewayID string, client qtt.Client, mqttMsg *models.MQTTMessage) error {
	return publishTelemetry(gatewayID, client, mqttMsg)
}

// Attaching a device requires qos = 2 (at most once, since it's noted in the chrysalis cloud datastore)
func AttachDeviceToGateway(gatewayID string, client qtt.Client, mqttMsg *models.MQTTMessage) error {
	return publishTelemetry(gatewayID, client, mqttMsg)
}

// Dettaching a device requires qos = 2
func DetachGatewayDevice(gatewayID string, client qtt.Client, mqttMsg *models.MQTTMessage) error {
	return publishTelemetry(gatewayID, client, mqttMsg)
}

// mqttLocalPublish publishing to redis pub/sub to be then forwarded to Chrysalis Cloud over MQTT protocol
func PublishToRedis(rdb *redis.Client, deviceID string, operation models.MQTTProcessOperation, processType string, customMessage []byte) error {
	// publish to chrysalis cloud the change
	pubSubMsg := &models.MQTTMessage{
		DeviceID:         deviceID,
		Created:          time.Now().UTC().Unix() * 1000,
		ProcessOperation: operation,
		ProcessType:      models.MQTTProcessType(processType),
		Message:          customMessage,
	}
	pubSubMsgBytes, imsgErr := json.Marshal(pubSubMsg)
	if imsgErr != nil {
		g.Log.Error("failed to publish redis internally", imsgErr)
		return imsgErr
	} else {
		rCmd := rdb.Publish(models.RedisLocalMQTTChannel, string(pubSubMsgBytes))
		if rCmd.Err() != nil {
			g.Log.Error("failed to publish change to redis internally", rCmd.Err())
			return rCmd.Err()
		}
	}
	return nil
}
