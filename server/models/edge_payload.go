package models

type EdgeCommandPayload struct {
	Type         string `json:"t"`              // type of the config payload (rtsp)
	Operation    string `json:"op"`             // a (add), r(remove), u(update)
	Name         string `json:"n"`              // name of the device on the edge
	RTSPEndpoint string `json:"rtsp,omitempty"` // rtsp endpoint
	RTMPEndpoint string `json:"rtmp,omitempty"` // rtmp endpoint
	ImageTag     string `json:"tag,omitempty"`  // image tag (e.g. chrysedgeproxy:0.0.7)
}
