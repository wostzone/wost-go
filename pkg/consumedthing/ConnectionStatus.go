package consumedthing

// ConnectionStatus contains the status of protocol bindings used in the factory
type ConnectionStatus struct {
	// Account that is connected
	//Account accounts.AccountRecord

	// AccessToken to authenticate with
	AccessToken string

	// Authenticated indicates the access token is valid
	Authenticated bool

	// AuthStatus with a text description of authentication result
	AuthStatus string

	// Connected indicates that a message bus is connected
	Connected bool

	// DirectoryRead indicates the TDs are obtained from the directory
	DirectoryRead bool

	// StatusMessage with a human description of the connection status
	StatusMessage string
}
