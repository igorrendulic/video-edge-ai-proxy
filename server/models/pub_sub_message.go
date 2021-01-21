package models

// PubSubMessage is used when requesting from gRPC (e.g. VideoBuffer)
type PubSubMessage struct {
	DeviceID      string `json:"deviceId"`
	FromTimestamp int64  `json:"fromTimestamp"`
	ToTimestamp   int64  `json:"toTimestamp"`
	RequestID     string `json:"requestId"`
}
