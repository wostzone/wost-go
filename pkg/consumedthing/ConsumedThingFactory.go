package consumedthing

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/wostzone/wost-go/pkg/accounts"
	"github.com/wostzone/wost-go/pkg/mqttclient"
	"github.com/wostzone/wost-go/pkg/thing"
	"github.com/wostzone/wost-go/pkg/tlsclient"
	"sync"
)

// ConsumedThingFactory for managing connected instances of consumed things.
// ConsumedThing's are created using the 'consume' method.
//
// This factory is intended for consumers for creating client side 'Thing' instances, eg consumed things
// It will bind the instance to protocol bindings for receiving and updating TD, property and events.
type ConsumedThingFactory struct {
	// accessToken to access Hub services. Use Authenticate() to refresh or login
	accessToken string

	// account used to access Hub services
	account *accounts.AccountRecord

	// authClient for obtaining authentication tokens. Not used when connecting with client cert.
	authClient *tlsclient.TLSClient

	// dirClient for querying the directory service
	dirClient *tlsclient.TLSClient

	// Bindings that are in use with consumed things by thing ID
	bindings map[string]*ConsumedThingProtocolBinding

	// CA certificate of server for validating the auth and directory services
	caCert *x509.Certificate

	// The current connection status of the factory bindings
	connectionStatus ConnectionStatus

	// Consumed things by thing ID
	ctMap map[string]*ConsumedThing

	// mutex for safe concurrent access to ctMap and bindings maps
	ctMapMutex sync.RWMutex

	// mqttClient holds the message bus connection
	mqttClient *mqttclient.MqttClient

	// store of TD documents
	thingStore *thing.ThingStore
}

// Authenticate or refresh the access token used by the authentication protocol.
//
// This does nothing if authenticated using a client certificate.
//
// If a password is provided then obtain a new access and refresh token pair using the account login ID.
// If no password is provided attempt to refresh the tokens.
func (ctFactory *ConsumedThingFactory) Authenticate(password string) error {
	var err error
	if ctFactory.authClient == nil {
		logrus.Infof("No auth client. Ignored ")
	} else if password != "" {
		logrus.Infof("With password. Attempt to get JWT tokens.")
		accessToken, err := ctFactory.authClient.ConnectWithJWTLogin(
			ctFactory.account.LoginName, password, "")
		if err == nil {
			ctFactory.accessToken = accessToken
		}
	} else {
		logrus.Infof("No password, attempt to refresh JWT tokens")
		tokens, err := ctFactory.authClient.RefreshJWTTokens("")
		if err == nil {
			ctFactory.accessToken = tokens.AccessToken
		}
	}
	return err
}

// Connect the factory to the Hub services and initialize the protocol binding connections.
//
// This will attempt to refresh the existing JWT token pair. If connect fails then retry using
// a valid password for the account.
//
// Call Disconnect before shutting down.
//
//  password to use when no valid refresh token is available
func (ctFactory *ConsumedThingFactory) Connect(password string) error {
	account := ctFactory.account

	// Shutdown existing connections
	ctFactory.Disconnect()

	logrus.Infof("account '%s' to: %s", account.ID, account.Address)

	//ctFactory.connectionStatus.Account = account
	ctFactory.connectionStatus.StatusMessage = ""
	ctFactory.thingStore = thing.NewThingStore(account.ID)
	ctFactory.thingStore.Load()

	// step 1: authenticate
	ctFactory.authClient.ConnectNoAuth()
	err := ctFactory.Authenticate(password)
	if err != nil {
		logrus.Errorf("Authentication failed. Retry with password.")
	} else {
		// step 2: Connect to the directory service in order to read TDs and values
		ctFactory.dirClient.ConnectWithJwtAccessToken(account.LoginName, ctFactory.accessToken)

		// step 3: connect to the mqtt message bus
		mqttHostPort := fmt.Sprintf("%s:%d", account.Address, account.MqttPort)
		err = ctFactory.mqttClient.ConnectWithAccessToken(mqttHostPort, account.LoginName, ctFactory.accessToken)
	}
	return err
}

// ConnectWithCert connects the factory to the Hub services using a client certificate
// for authentication.
//
// Call Disconnect before shutting down.
//
//  clientCert with the certificate signed by the Hub CA
func (ctFactory *ConsumedThingFactory) ConnectWithCert(clientCert *tls.Certificate) error {
	account := ctFactory.account

	// Shutdown existing connections
	ctFactory.Disconnect()

	logrus.Infof("account '%s' to: %s", account.ID, account.Address)

	//ctFactory.connectionStatus.Account = account
	ctFactory.connectionStatus.StatusMessage = ""
	ctFactory.thingStore = thing.NewThingStore(account.ID)
	ctFactory.thingStore.Load()

	// connecting to the message bus is mandatory.
	// Things still function if the directory service cannot be reached
	mqttHostPort := fmt.Sprintf("%s:%d", account.Address, account.MqttPort)
	err := ctFactory.mqttClient.ConnectWithClientCert(mqttHostPort, clientCert)
	if err == nil {
		err = ctFactory.dirClient.ConnectWithClientCert(clientCert)
	}
	return err
}

// Consume returns a 'Consumed Thing' instance for interacting with a remote (exposed) thing and binds it
// to the relevant protocol bindings. This is the only method allowed to create consumed thing instances.
//
// This attaches it to interaction protocol bindings:
// - directory binding to read properties and history
// - mqtt binding to subscribe and request updates
//
// If a consumed thing already exists then simply return it.
//
// @param td is the Thing TD whose interaction instance to create
func (ctFactory *ConsumedThingFactory) Consume(td *thing.ThingTD) *ConsumedThing {
	logrus.Infof("Thing: %s", td.ID)

	ctFactory.ctMapMutex.Lock()
	defer ctFactory.ctMapMutex.Unlock()
	cThing, found := ctFactory.ctMap[td.ID]

	if !found {
		// WoST communication is mqtt and http based
		cThing = CreateConsumedThing(td)
		binding := CreateConsumedThingProtocolBinding(cThing)
		ctFactory.bindings[td.ID] = binding
		ctFactory.ctMap[td.ID] = cThing
		binding.Start(
			ctFactory.authClient,
			ctFactory.dirClient,
			ctFactory.mqttClient)
	}
	return cThing
}

// Destroy stops and removes the consumed thing.
// This stops listening to external events
func (ctFactory *ConsumedThingFactory) Destroy(cThing *ConsumedThing) {
	logrus.Infof("Thing: %s", cThing.TD.ID)
	ctFactory.ctMapMutex.Lock()
	defer ctFactory.ctMapMutex.Unlock()

	// stop and remove the consumed thing protocol binding
	binding := ctFactory.bindings[cThing.TD.ID]
	if binding != nil {
		binding.Stop()
		delete(ctFactory.bindings, cThing.TD.ID)
	}

	// stop and remove the consumed thing instance
	cThing.Stop()
	delete(ctFactory.ctMap, cThing.TD.ID)
}

// Disconnect the factory from the account
func (ctFactory *ConsumedThingFactory) Disconnect() {
	if ctFactory.mqttClient != nil {
		ctFactory.mqttClient.Disconnect()
	}
	ctFactory.connectionStatus.Connected = false
	if ctFactory.thingStore != nil {
		ctFactory.thingStore.Save()
	}
}

// GetThingStore returns the Thing store where the factory keeps its things
func (ctFactory *ConsumedThingFactory) GetThingStore() *thing.ThingStore {
	return ctFactory.thingStore
}

// CreateConsumedThingFactory creates a factory instance for consumed things for the given account
//
// If no CA certificate is provided there will be no protection against a man-in-the-middle attack.
// To obtain a CA, request it from the administrator, copy it from the Hub, copy it from the web browser
// or use the idprov service, depending on the circumstances.
//
//  appID unique ID of the application instance
//  account used to connect with
//  caCert previously obtained CA certificate used to validate the server
func CreateConsumedThingFactory(
	appID string, account *accounts.AccountRecord, caCert *x509.Certificate) *ConsumedThingFactory {

	authHostPort := fmt.Sprintf("%s:%d", account.Address, account.AuthPort)
	dirHostPort := fmt.Sprintf("%s:%d", account.Address, account.DirectoryPort)
	//mqttHostPort := fmt.Sprintf("%s:%d", account.Address, account.MqttPort)

	ctFactory := &ConsumedThingFactory{
		account:    account,
		bindings:   make(map[string]*ConsumedThingProtocolBinding),
		ctMap:      make(map[string]*ConsumedThing),
		ctMapMutex: sync.RWMutex{},
		thingStore: thing.NewThingStore(""),
		//
		authClient: tlsclient.NewTLSClient(authHostPort, caCert),
		dirClient:  tlsclient.NewTLSClient(dirHostPort, caCert),
		mqttClient: mqttclient.NewMqttClient(appID, caCert, 0),
	}
	return ctFactory
}
