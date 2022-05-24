// Package consumedthing that implements the ConsumedThing API
// Consumed Things are remote representations of Things used by consumers.
package consumedthing

import (
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/wostzone/wost-go/pkg/thing"
	"sync"
)

// ConsumedThing is the implementation of an ConsumedThing interface
// This is modelled after the scripting definition of [WoT Consumed Thing](https://w3c.github.io/wot-scripting-api/#the-consumedthing-interface).
// It attempts to be as similar as possible but language differences will cause some differences. That said,
// the concepts and most of its usage will be similar.
//
// Key differences:
//  1. no JS 'Promises'
//  2. consumed things cache their properties so readproperty will obtain immediate results
//
type ConsumedThing struct {

	/** Hook to invoke action via the protocol binding.
	 * This can be set to a protocol binding by the protocol factory
	 * By default this throws an error
	 *
	 * @param cThing of the thing whose action to invoke
	 * @param name of the action to invoke
	 * @param params containing the data of the action as defined in the action affordance schema
	 * @returns an error or nil
	 */
	InvokeActionHook func(name string, params interface{}) error

	/** Hook to refresh the cashed property values via the protocol binding.
	 * This can be set to a protocol binding by the protocol factory
	 * By default this throws an error
	 *
	 * @param cThing of the thing whose properties to refresh
	 * @returns a promise that resolves when the request to read properties has been sent
	 */
	//ReadPropertiesHook func() error

	/** Hook to write properties via the protocol binding
	 * This can be set to a protocol binding by the protocol factory.
	 * By default this throws an error.
	 *
	 * @param cThing of the thing to write to
	 * @param props containing the name-value pair where value is the text representation to write.
	 * @returns a promise that resolves when the request to write properties has been sent
	 */
	WritePropertyHook func(propName string, propValue any) error

	// internal slot for subscriptions to property changes
	activeObservations map[string]Subscription
	// internal slot for subscriptions to events
	activeSubscriptions map[string]Subscription
	// mutex for async updating of subscriptions
	subscriptionMutex sync.Mutex

	// Internal slot with Thing Description document this consumed thing consumes
	TD *thing.ThingTD

	// valueStore holds the last property and event values
	valueStore map[string]*thing.InteractionOutput
	// mutex for concurrent access to stored values
	valueStoreMutex sync.RWMutex
}

// _getValue reads the latest cached value from the value store
// This is concurrent safe and should be the only way to access the values.
func (cThing *ConsumedThing) _getValue(key string) (value *thing.InteractionOutput, found bool) {
	cThing.valueStoreMutex.RLock()
	defer cThing.valueStoreMutex.RUnlock()
	value, found = cThing.valueStore[key]
	return value, found
}

// _putValue writes the latest value into the value store cache
// This is concurrent safe and should be the only way to access the values.
func (cThing *ConsumedThing) _putValue(key string, value *thing.InteractionOutput) {
	logrus.Infof("Updating value of %s", key)
	cThing.valueStoreMutex.Lock()
	defer cThing.valueStoreMutex.Unlock()
	cThing.valueStore[key] = value
}

// GetThingDescription returns the TD document of this consumed Thing
// This returns the cached version of the TD
func (cThing *ConsumedThing) GetThingDescription() *thing.ThingTD {
	return cThing.TD
}

// HandleEvent handles incoming events for the consumed thing.
//
// If the event name is 'properties' then the payload is a map of property name-value pairs.
// If the event is a propertyName then the payload is the property value of that event.
//
// Otherwise the event payload is described in the TD event affordance.
// Last invoke the subscriber to the event name, if any, or the default subscriber
//  address is the MQTT topic that the event is published on as: things/{thingID}/event/{eventName}
//  whereas message is the body of the event.
func (cThing *ConsumedThing) HandleEvent(eventName string, message []byte) {
	var evData *thing.InteractionOutput

	logrus.Infof("Received event '%s' for thing: %s", eventName, cThing.TD.ID)

	// handle property events
	propAffordance := cThing.TD.GetProperty(eventName)
	if propAffordance != nil {
		evData = thing.NewInteractionOutputFromJson(message, &propAffordance.DataSchema)

		logrus.Infof("Event with name %s is a property event; for thing: %s", eventName, cThing.TD.ID)
		// TODO validate the data
		// property or event, it is stored in the valueStore
		cThing._putValue(eventName, evData)

		// notify observer if any
		subscription, found := cThing.activeObservations[eventName]
		if found {
			subscription.Handler(eventName, evData)
		}
	} else if eventName == TopicSubjectProperties {
		// handle map of property name-value pairs
		var propMap map[string]interface{}
		err := json.Unmarshal(message, &propMap)
		if err != nil {
			logrus.Warningf("Event with name '%s' does not contain name-value map for thing: %s",
				eventName, cThing.TD.ID)
			return
		}
		for propName, propValue := range propMap {
			propAffordance = cThing.TD.GetProperty(propName)
			if propAffordance != nil {
				evData = thing.NewInteractionOutput(propValue, &propAffordance.DataSchema)
				// property or event, it is stored in the valueStore
				cThing._putValue(propName, evData)

				// notify observer if any
				subscription, found := cThing.activeObservations[propName]
				if found {
					subscription.Handler(propName, evData)
				}
			} else {
				logrus.Infof("Ignoring unknown property '%s' for thing: %s",
					propName, cThing.TD.ID)
			}
		}
	}
	// handle actual events
	eventAffordance := cThing.TD.GetEvent(eventName)
	if eventAffordance != nil {
		evData = thing.NewInteractionOutputFromJson(message, &eventAffordance.Data)
		// property or event, it is stored in the valueStore
		cThing._putValue(eventName, evData)

		// notify subscriber if any
		subscription, found := cThing.activeSubscriptions[eventName]
		if found {
			subscription.Handler(eventName, evData)
		}
	}
}

// InvokeAction makes a request for invoking an Action and returns once the
// request is submitted.
//
// This will be posted on topic: "things/{thingID}/action/{actionName}" with data as payload
//
// Takes as arguments actionName, optionally action data as defined in the TD.
// Returns nil if the action request was submitted successfully or an error if failed
func (cThing *ConsumedThing) InvokeAction(actionName string, data interface{}) error {
	aa := cThing.TD.GetAction(actionName)
	if aa == nil {
		err := errors.New("can't invoke action '" + actionName +
			"'. Action is not defined in TD '" + cThing.TD.ID + "'")
		logrus.Error(err)
		return err
	}
	if cThing.InvokeActionHook == nil {
		err := errors.New("Missing hook for action: " + actionName)
		logrus.Error(err)
		return err
	}
	return cThing.InvokeActionHook(actionName, data)
}

// ObserveProperty makes a request for Property value change notifications.
// Takes as arguments propertyName and a handler.
//
// returns an error if an active observation already exists
func (cThing *ConsumedThing) ObserveProperty(
	name string, handler func(name string, data *thing.InteractionOutput)) error {
	var err error = nil

	// Only a single subscriber is allowed
	_, found := cThing.activeObservations[name]
	if found {
		logrus.Errorf("A property subscription for '%s' already exists", name)
		return errors.New("NotAllowed")
	}

	sub := Subscription{
		SubType: SubscriptionTypeProperty,
		Name:    name,
		Handler: handler,
	}
	cThing.activeObservations[name] = sub
	return err
}

// ReadProperty reads a Property value from the local cache.
// Returns the last known property value or an error if the name is not a known property.
func (cThing *ConsumedThing) ReadProperty(name string) (*thing.InteractionOutput, error) {
	//return res, errors.New("'"+name + "' is not a known property" )
	value, found := cThing._getValue(name)
	if !found {
		// TODO: property exists but there is no known value
		return value, errors.New("Property " + name + " does not exist on thing " + cThing.TD.ID)
	}
	return value, nil
}

// ReadMultipleProperties reads multiple Property values with one request.
// propertyNames is an array with names of properties to return
// Returns a PropertyMap object that maps keys from propertyNames to InteractionOutput of that property.
func (cThing *ConsumedThing) ReadMultipleProperties(names []string) map[string]*thing.InteractionOutput {
	res := make(map[string]*thing.InteractionOutput, 0)
	for _, name := range names {
		output, _ := cThing.ReadProperty(name)
		res[name] = output
	}
	return res
}

// ReadAllProperties reads all properties of the Thing with one request.
// Returns a PropertyMap object that maps keys from all Property names to InteractionOutput
// of the properties.
func (cThing *ConsumedThing) ReadAllProperties() map[string]*thing.InteractionOutput {
	res := make(map[string]*thing.InteractionOutput, 0)

	for name := range cThing.TD.Properties {
		output, _ := cThing.ReadProperty(name)
		res[name] = output
	}
	return res
}

// Stop delivering notifications for event subscriptions
// This is an internal method for use by the factory.
func (cThing *ConsumedThing) Stop() {
	cThing.subscriptionMutex.Lock()
	defer cThing.subscriptionMutex.Unlock()
	cThing.activeSubscriptions = make(map[string]Subscription)
	cThing.activeObservations = make(map[string]Subscription)

	cThing.valueStoreMutex.Lock()
	defer cThing.valueStoreMutex.Unlock()
	cThing.valueStore = make(map[string]*thing.InteractionOutput)
}

// SubscribeEvent makes a request for subscribing to events
//
// Takes as arguments eventName, listener
// Returns nil if subscription is successful or NotAllowed error if a subscription already exists
func (cThing *ConsumedThing) SubscribeEvent(
	eventName string, handler func(eventName string, data *thing.InteractionOutput)) error {
	cThing.subscriptionMutex.Lock()
	defer cThing.subscriptionMutex.Unlock()

	// Only a single subscriber is allowed
	_, found := cThing.activeSubscriptions[eventName]
	if found {
		logrus.Errorf("A subscription to event '%s' already exists", eventName)
		return errors.New("NotAllowed")
	}
	// the TD fragment describing the interaction
	eventAffordance := cThing.TD.GetEvent(eventName)

	sub := Subscription{
		SubType:     SubscriptionTypeEvent, // what is the purpose of capturing this?
		Name:        eventName,
		Handler:     handler,
		interaction: eventAffordance,
	}
	cThing.activeSubscriptions[eventName] = sub
	return nil
}

// WriteProperty submit a request to change a property value.
// Takes as arguments propertyName and value, and sends a property update to the exposedThing that in turn
// updates the actual device.
// This does not update the property immediately. It is up to the exposedThing to perform necessary validation
// and notify subscribers with an event after the change has been applied.
//
// There is no error feedback in case the request cannot be handled. The requester will only receive a
// property change event when the request has completed successfully. Failure to complete the request can be caused
// by an invalid value or if the IoT device is not in a state to accept changes.
//
// TBD: if there is a need to be notified of failure then a future update can add a write-property failed event.
//
// This will be published on topic "things/{thingID}/action/{name}"
//
// It returns an error if the property update could not be sent and nil if it is successfully
//  published. Final confirmation is obtained if an event is received with the updated property value.
func (cThing *ConsumedThing) WriteProperty(propName string, value interface{}) error {
	if cThing.WritePropertyHook == nil {
		return errors.New("WriteProperty is not supported for ConsumedThing. No hook is installed")
	}
	return cThing.WritePropertyHook(propName, value)
}

// WriteMultipleProperties writes multiple property values.
// Takes as arguments properties - as a map keys being Property names and values as Property values.
//
// This will be posted as individual update requests
//
// It returns an error if the action could not be sent and nil if the action is successfully
//  published. Final success is achieved if the property value will be updated through an event.
func (cThing *ConsumedThing) WriteMultipleProperties(properties map[string]interface{}) error {
	var err error

	for propName, value := range properties {
		err = cThing.WriteProperty(propName, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateConsumedThing constructs a consumed thing from a TD.
//
// A consumed Thing is a remote instance of a thing for the purpose of interaction with thing providers.
// This is intended for use by the ConsumedThingFactory only.
// The factory installs the hooks to connect to a protocol binding for read/write
//
// Use factory.consume() to obtain a working instance.
// @param td is a Thing Description document of the Thing to consume.
func CreateConsumedThing(td *thing.ThingTD) *ConsumedThing {
	cThing := &ConsumedThing{
		//readPropertiesHook: mb.ReadProperties,
		activeSubscriptions: make(map[string]Subscription),
		activeObservations:  make(map[string]Subscription),
		subscriptionMutex:   sync.Mutex{},
		valueStoreMutex:     sync.RWMutex{},
		TD:                  td,
		valueStore:          make(map[string]*thing.InteractionOutput),
	}
	return cThing
}
