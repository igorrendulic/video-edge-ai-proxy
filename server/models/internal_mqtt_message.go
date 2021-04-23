package models

type MQTTProcessOperation string
type MQTTProcessType string

const (
	DeviceOperationAdd      string = "add"           // add device
	DeviceOperationRemove   string = "remove"        // remove device
	DeviceOperationState    string = "state"         // the current state of device
	DeviceOperationStats    string = "stats"         // device stats (host system and each device)
	GatewayOperationCheckIn string = "gwcheckin"     // gateway checkin
	DeviceInternalTesting   string = "internal_test" // internal development event. Not used or required for regular operations (TODO: movw to unit tests)

	DeviceOperationStart  string = "start"  // start a new device on the edge
	DeviceOperationDelete string = "delete" // delete the device from the edge

	DeviceOperationUpgradeAvailable string = "upgrade_avail" // device has an upgrade available
	DeviceOperationUpgradeFinished  string = "upgrade"       // device has performed an upgrade

	DeviceOperationError string = "error" // device operation failed

	ProcessTypeRTSP        string = "rtsp"
	ProcessTypeApplication string = "app"
	ProcessTypeStats       string = "stats"
	ProcessTypeUnknown     string = "unknown"
)

// InternalMQTTMessage is a message within local redis system to respond to changes such as adding/removing camera
type MQTTMessage struct {
	DeviceID         string               `json:"deviceId"`                 // for which device the internal message is
	Created          int64                `json:"created,omitempty"`        // time of creation of the message
	ImageTag         string               `json:"imageTag,omitempty"`       // docker image tag
	RTMPEndpoint     string               `json:"rtmpEndpoint,omitempty"`   // possible rtmp endpoint
	RTSPConnection   string               `json:"rtspConnection,omitempty"` // rtspConnection string
	ProcessOperation MQTTProcessOperation `json:"operation"`                // the state of the internal process
	ProcessType      MQTTProcessType      `json:"type,omitempty"`           // type of internal process
	State            string               `json:"state,omitempty"`          // process state (running, restarting, ...)
	Message          []byte               `json:"message,omitempty"`        // optional custom message
}
