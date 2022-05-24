package exposedthing

import (
	"crypto/x509"
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
// This factory is intended for IoT devices to creating server side 'Thing' instances.
// It will bind the instance to protocol bindings for publishing TDs, properties and events,
// and receive action and property change requests as sent by consumed things.
type ExposedThingFactory struct {
	// Bindings that are in use with exposed things by thing ID
	bindings map[string]*ExposedThingMqttBinding

	// CA certificate for validating the message bus broker
	caCert *x509.Certificate

	// Exposed things by thing ID
	etMap map[string]*ExposedThing

	// mutex for safe concurrent access to etMap and bindings maps
	etMapMutex sync.RWMutex

	// mqttClient holds the message bus connection
	mqttClient *mqttclient.MqttClient
}

// Destroy stops and removes the exposed thing.
// This stops listening to external requests.
func (etFactory *ExposedThingFactory) Destroy(eThing *ExposedThing) {
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
