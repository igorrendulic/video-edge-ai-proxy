package models

type PubSubMessage struct {
	DeviceID      string `json:"deviceId"`
	FromTimestamp int64  `json:"fromTimestamp"`
	ToTimestamp   int64  `json:"toTimestamp"`
	RequestID     string `json:"requestId"`
}
