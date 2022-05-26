package exposedthing

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/wostzone/wost-go/pkg/mqttclient"
	"github.com/wostzone/wost-go/pkg/thing"
	"sync"
)

// Factory for managing instances of exposed things

/** UNDER CONSTRUCTION **/

// ExposedThingFactory for managing connected instances of exposed things.
// Exposed Things are created using the 'expose' method.
// (the spec also mentions 'produce' and 'exposeInit'. Not clear how that is supposed to work so lets keep it simple.
//
// This factory is intended for IoT devices to produce and expose 'Thing' instances to consumers.
// It will bind the instance to protocol bindings for publishing TDs, properties and events,
// and receive action and property change requests as sent by consumed things.
type ExposedThingFactory struct {
	// Bindings that are in use with exposed things by thing ID
	bindings map[string]*ExposedThingMqttBinding

	// CA certificate for validating the message bus broker
	caCert *x509.Certificate

	// Client certificate used to authenticate
	clientCert *tls.Certificate

	// Exposed things by thing ID
	etMap map[string]*ExposedThing

	// mutex for safe concurrent access to etMap and bindings maps
	etMapMutex sync.RWMutex

	// mqttClient holds the message bus connection
	mqttClient *mqttclient.MqttClient
}

// Connect the factory to message bus.
// This uses the client certificate for authentication.
//  address of the hub server that runs the mqtt broker
//  mqttPort with port of the mqtt broker for certificate auth
func (etFactory *ExposedThingFactory) Connect(address string, mqttPort int) error {
	logrus.Infof("address=%s, mqttPort=%d", address, mqttPort)
	hostPort := fmt.Sprintf("%s:%d", address, mqttPort)
	return etFactory.mqttClient.ConnectWithClientCert(hostPort, etFactory.clientCert)
}

// Disconnect the factory from the message bus
func (etFactory *ExposedThingFactory) Disconnect() {
	logrus.Infof("")
	if etFactory.mqttClient != nil {
		etFactory.mqttClient.Disconnect()
	}
}

// Destroy stops and removes the exposed thing.
// This stops listening to external requests.
func (etFactory *ExposedThingFactory) Destroy(eThing *ExposedThing) {
	logrus.Infof("exposed thing: %s", eThing.TD.ID)
	etFactory.etMapMutex.Lock()
	defer etFactory.etMapMutex.Unlock()

	// stop and remove the protocol binding
	binding := etFactory.bindings[eThing.TD.ID]
	if binding != nil {
		binding.Stop()
		delete(etFactory.bindings, eThing.TD.ID)
	}

	// stop and remove the consumed thing instance
	eThing.Destroy()
	delete(etFactory.etMap, eThing.TD.ID)
}

// Expose creates an exposed thing instance and starts serving external requests for the Thing so that
// WoT Interactions using Properties and Actions will be possible.
// This also publishes the TD document of this Thing.
func (etFactory *ExposedThingFactory) Expose(deviceID string, td *thing.ThingTD) *ExposedThing {
	logrus.Infof("device '%s'; ID: %s", deviceID, td.ID)

	etFactory.etMapMutex.Lock()
	defer etFactory.etMapMutex.Unlock()
	eThing, found := etFactory.etMap[td.ID]

	if !found {
		eThing = CreateExposedThing(deviceID, td)
		binding := CreateExposedThingMqttBinding(eThing, etFactory.mqttClient)
		etFactory.bindings[td.ID] = binding
		etFactory.etMap[td.ID] = eThing
		binding.Start()
	}
	return eThing
}

// CreateExposedThingFactory creates a factory instance for exposed things.
//
// Intended for use by IoT devices and Hub services. IoT devices authenticate themselves with a client certificate
// obtained during the provisioning process using the idprov client.
// Hub services have access to the hub service TLS certificate configured in the Hub configuration file.
//
//  appID unique ID of the application instance
//  clientCert with the certificate for authentication
//  caCert previously obtained CA certificate used to validate the server
func CreateExposedThingFactory(
	appID string, clientCert *tls.Certificate, caCert *x509.Certificate) *ExposedThingFactory {

	//authHostPort := fmt.Sprintf("%s:%d", account.Address, account.AuthPort)
	//dirHostPort := fmt.Sprintf("%s:%d", account.Address, account.DirectoryPort)
	//mqttHostPort := fmt.Sprintf("%s:%d", account.Address, account.MqttPort)

	etFactory := &ExposedThingFactory{
		bindings:   make(map[string]*ExposedThingMqttBinding),
		clientCert: clientCert,
		etMap:      make(map[string]*ExposedThing),
		etMapMutex: sync.RWMutex{},
		//
		mqttClient: mqttclient.NewMqttClient(appID, caCert, 0),
	}
	return etFactory
}
