package models

// EdgeConnectCredentials returned edge connection credentials from Chrysalis Cloud
type EdgeConnectCredentials struct {
	ID            string `json:"keyId,omitempty"` // edge key id
	PrivateKeyPem []byte `json:"privateKeyPem"`   // pem private key for gateway to connect to cloud
	RegistryID    string `json:"registryId"`      // registry ID
	ProjectID     string `json:"projectId"`       // project ID
	GatewayID     string `json:"gatewayId"`       // gatewayID
	Region        string `json:"region"`          // region of the registry and gateway
}
