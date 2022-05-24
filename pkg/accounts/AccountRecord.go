package accounts

// AccountRecord holds the account information of a WoST Hub
type AccountRecord struct {
	// Address of the hub or "" to use auto-discovery
	Address string `json:"address"`

	// ID holds the account unique identifier
	ID string `json:"id"`

	// DisplayName holds the friendly display name
	DisplayName string `json:"name"`

	// LoginName with the name used to login to the hub
	LoginName string `json:"loginName"`

	// AuthPort with the authentication service port. Default is 8881. 0 to use the node express proxy server
	AuthPort int `json:"authPort"`

	// DirectoryPort to connect to the directory service. Default is 8886. 0 to use the node express server
	DirectoryPort int `json:"directoryPort"`

	// MqttPort to connect with the MQTT broker. Default is 8885 for websocket, 8883 for TCP or 8884 for certificate
	MqttPort int `json:"authPort"`

	// Enabled to try to use this connection
	Enabled bool `json:"enabled"`

	// RememberMe to remember the refresh token to avoid asking for login for this account
	RememberMe bool `json:"rememberMe"`
}
