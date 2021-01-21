package models

type MQTTProcessOperation string
type MQTTProcessType string

const (
	DeviceOperationStart   string = "start"
	DeviceOperationStop    string = "stop"
	DeviceOperationUpgrade string = "upgrade"

	ProcessTypeDevice string = "device"
)

// InternalMQTTMessage is a message within local redis system to respond to changes such as adding/removing camera
type MQTTMessage struct {
	DeviceID         string               `json:"deviceId"`          // for which device the internal message is
	Created          int64                `json:"created"`           // time of creation of the message
	ImageTag         string               `json:"imageTag"`          // docker image tag
	ProcessOperation MQTTProcessOperation `json:"operation"`         // the state of the internal process
	ProcessType      MQTTProcessType      `json:"type"`              // type of internal process (currently only for device -> aka camera)
	State            string               `json:"state"`             // process state (running, restarting, ...)
	Message          string               `json:"message,omitempty"` // optional custom message
}
