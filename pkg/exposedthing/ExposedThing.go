// Package exposedthing that implements the ExposedThing API
// Exposed Things are used by IoT device implementers to provide access to the device.
package exposedthing

import (
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/wostzone/wost-go/pkg/thing"
	"sync"
)

// ExposedThing is the implementation of an ExposedThing interface using the MQTT protocol binding.
// Thing implementers can use this API to subscribe to actions and publish TDs and events.
//
// This loosely follows the WoT scripting API for ExposedThing as described at
// https://www.w3.org/TR/wot-scripting-api/#the-exposedthing-interface
//
// Differences with the WoT scripting API:
//  1. The WoT scripting API uses ECMAScript with promises for asynchronous results.
//     This implementation uses channels to return async results, which is golang idiomatic.
//  2. The WoT scripting API method names are in 'lowerCase' format.
//     In golang lowerCase makes things private. This implementation uses 'UpperCase' name format.
//  3. Most methods are synchronous instead of asynchronous as the MQTT client is synchronous.
//     The result of actions indicates that it was submitted successfully. Actions do not have
//     a return value (in WoST) as they are not remote procedure calls. If the effect of an
//     action is needed then consumers should subscribe to property changes that are submitted
//     as the action is executed. The results of actions by others then be handled in the same way.
//  4. Actions are only handled by devices that are not asleep as the message bus does not
//     yet support queued actions. This is a limitation of the message bus. Future implementations
//     of the message bus can add queuing to support intermittent connected devices.
//     Use of the 'retain' flag is not recommended for actions on devices that also have a manual input.
//  5. If an action is not allowed then no error is returned. In most cases the MQTT bus won't accept
//     the request in the first place.
//  6. Additional functions UpdatePropertyValue(s) to support sending property change events only
//     when property values change.
//
// Example of properties in a TD with forms for mqtt protocol binding.
// The forms will likely be provided through a @context link to a standardized WoST model, once the semantics are figured out.
// {
//   "properties": {
//        "onoff": {
//            "@type": "iot:SwitchOnOff",
//            "title": "Switch on or off status"
//            "description": "More elaborate description of the onoff property"
//            "observable": true,    // form must provide an observeproperty binding
//            "type": "boolean",
//            "unit": "binary",
//            "readOnly": false,  // property is writable. form must provide a writeproperty binding
//        }
//        // These forms apply to all writable properties
//        "forms": [{
//          	"op": ["writeproperty", "writeproperties"],
//          	"href": "mqtts://{broker}/things/{thingID}/write/properties",
//				"mqv:controlPacketValue": "PUBLISH",
//              "contentType": "application/json"
//          }, {
//              // TBD. MQTT topic. How to parameterize in a generic schema?
//              "op": ["observeproperty"],
//          	"href": "mqtts://{broker}/things/{thingID}/event/properties",
//				"mqv:controlPacketValue": "SUBSCRIBE",
//              "contentType": "application/json"
//         }],
//       }
//    }
// }

type ExposedThing struct {

	// deviceID for reverse looking of device by their internal ID
	DeviceID string

	// Protocol binding hook to emit an event
	EmitEventHook func(name string, data interface{}) error

	// Protocol binding hook to emit properties changed notification
	EmitPropertiesChangeHook func(map[string]interface{}) error

	// handler for action requests
	// to set the default handler use name ""
	actionHandlers map[string]func(eThing *ExposedThing, actionName string, value *thing.InteractionOutput) error

	// mutex for async updating of action and property handlers
	handlerMutex sync.RWMutex

	// handler for writing property requests
	// to set the default handler use name ""
	propertyWriteHandlers map[string]func(eThing *ExposedThing, propName string, value *thing.InteractionOutput) error

	// Internal slot with Thing Description document this exposed thing exposes
	TD *thing.ThingTD

	// valueStore holds the last property and event values
	valueStore map[string]*thing.InteractionOutput

	// mutex for concurrent access to stored values
	valueStoreMutex sync.RWMutex
}

// _getValue reads the latest cached value from the value store
// This is concurrent safe and should be the only way to access the values.
func (eThing *ExposedThing) _getValue(key string) (value *thing.InteractionOutput, found bool) {
	eThing.valueStoreMutex.RLock()
	defer eThing.valueStoreMutex.RUnlock()
	value, found = eThing.valueStore[key]
	return value, found
}

// _putValue writes the latest value into the value store cache
// This is concurrent safe and should be the only way to access the values.
func (eThing *ExposedThing) _putValue(key string, value *thing.InteractionOutput) {
	eThing.valueStoreMutex.Lock()
	defer eThing.valueStoreMutex.Unlock()
	eThing.valueStore[key] = value
}

// Destroy stops serving external requests
// this is an internal method for use by the factory
func (eThing *ExposedThing) Destroy() {
	eThing.handlerMutex.Lock()
	defer eThing.handlerMutex.Unlock()
	eThing.propertyWriteHandlers = nil
	eThing.actionHandlers = nil
}

// EmitEvent publishes a single event to subscribers.
// This invokes the EmitEventHook from the protocol binding
//
// name is the name of the event as described in the TD, or one of the general purpose events.
// data is the event value as defined in the TD events schema and used as the payload
// Returns an error if the event is not found or cannot be published
func (eThing *ExposedThing) EmitEvent(name string, data interface{}) error {
	var err error
	_, found := eThing.TD.Events[name]
	if !found {
		logrus.Errorf("EmitEvent. Event '%s' not defined for thing '%s'", name, eThing.TD.ID)
		err = errors.New("NotFoundError")
	} else if eThing.EmitEventHook == nil {
		logrus.Errorf("EmitEvent. EmitEventHook is not installed for thing %s", eThing.TD.ID)
		err = errors.New("EmitEventHook not installed error")
	} else {
		err = eThing.EmitEventHook(name, data)
	}
	return err
}

// EmitPropertyChange publishes a property value change event, which in turn will notify all
// observers (subscribers) of the change.
//
// propName is the name of the property in the TD.
// newRawValue is the new raw value of the property. This will be also be stored in the valueStore.
// Returns an error if the property value cannot be published
func (eThing *ExposedThing) EmitPropertyChange(propName string, newRawValue interface{}) error {
	propMap := map[string]interface{}{propName: newRawValue}
	return eThing.EmitPropertiesChange(propMap, false)
}

// EmitPropertiesChange sends a properties change event for multiple properties
// This will remove properties that do not have an affordance.
//
// This invokes the EmitPropertiesChangeHook from the protocol binding and updates the cached
// value.
//
// For property names that are defined as events, an event is sent for each property in the event list.
//
// @param onlyChanges: include only those properties whose value have changed (recommended)
// Returns an error if submitting an event fails
func (eThing *ExposedThing) EmitPropertiesChange(
	propMap map[string]interface{}, onlyChanges bool) error {
	//logrus.Infof("EmitPropertyChanges: %s", propMap)
	var err error
	changedProps := make(map[string]interface{})

	// filter properties that have no affordance or haven't changed
	for propName, newVal := range propMap {
		lastVal, found := eThing._getValue(propName)

		// In order to be included as a property it must have a propertyAffordance
		if !found || !onlyChanges || lastVal.Value != newVal {
			propAffordance, found := eThing.TD.Properties[propName]
			// only include values that are in the properties map
			if found {
				changedProps[propName] = newVal
				newIO := thing.NewInteractionOutput(newVal, &propAffordance.DataSchema)
				eThing._putValue(propName, newIO)
			}
			//
			//// to be sent as an event it must have an event affordance
			//eventAffordance, found := eThing.TD.Events[propName]
			//if found {
			//	_ = eventAffordance
			//	topic := strings.ReplaceAll(TopicEmitEvent, "{thingID}", eThing.TD.ID)
			//	topic += "/" + propName
			//	err = eThing.mqttClient.PublishObject(topic, newVal)
			//	if err != nil {
			//		logrus.Warningf("MqqExposedThing.EmitPropertyChanges: Failed %s", err)
			//		return err
			//	}
			//}
		}
	}
	// only publish if there are properties left
	if len(changedProps) > 0 {
		err = eThing.EmitPropertiesChangeHook(changedProps)
	}
	return err
}

// GetThingDescription returns the TD document of this exposed Thing
// This returns the cached version of the TD
func (eThing *ExposedThing) GetThingDescription() *thing.ThingTD {
	return eThing.TD
}

// Expose starts serving external requests for the Thing so that WoT Interactions using Properties and Actions
// will be possible. This also publishes the TD document of this Thing.
//func (eThing *ExposedThing) Expose() error {
//	// Actions and Properties are handled the same.
//	// An action with a property name will update the property.
//	topic := strings.ReplaceAll(TopicInvokeAction, "{thingID}", eThing.TD.ID) + "/#"
//	eThing.mqttClient.Subscribe(topic, eThing.HandleActionRequest)
//
//	// Also publish this Thing's TD document
//	topic = strings.ReplaceAll(TopicThingTD, "{thingID}", eThing.TD.ID)
//	err := eThing.mqttClient.PublishObject(topic, eThing.TD)
//	return err
//}

// HandleActionRequest for this Thing to be invoked by the protocol binding.
// This passes the request to the registered action handler.
// If no specific handler is set then the default handler with name "" is invoked.
func (eThing *ExposedThing) HandleActionRequest(actionName string, message []byte) {
	var actionData *thing.InteractionOutput
	var err error

	logrus.Infof("actionName '%s', message: '%s'", actionName, message)

	// TODO: Are channels a better way for the protocol binding to push action requests? do we care?

	// determine the action schema
	actionAffordance := eThing.TD.GetAction(actionName)
	if actionAffordance != nil {
		// this is a registered action
		actionData = thing.NewInteractionOutputFromJson(message, &actionAffordance.Input)
		// TODO validate the data against the schema

		// action specific handlers takes precedence
		handler, _ := eThing.actionHandlers[actionName]
		if handler != nil {
			err = handler(eThing, actionName, actionData)
		} else {
			// default handler is a fallback
			defaultHandler, _ := eThing.actionHandlers[""]
			if defaultHandler != nil {
				err = defaultHandler(eThing, actionName, actionData)
			} else {
				err = errors.New("no handler for action request")
			}
		}
	} else {
		// properties are written using actions
		eThing.handlePropertyWriteRequest(actionName, message)
	}
	if err != nil {
		logrus.Errorf("Request failed for topic %s: %s", actionName, err)
	}
}

// handlePropertyWriteRequest for updating a property
// This invokes the property update handler with the value of the new property.
//
// It is up to the handler to invoke emitPropertyChange after the change has been applied.
//
// There is currently no error feedback in case the request cannot be handled. The requester will receive a
// property change event when the request has completed successfully.
//
// TBD: if there is a need to be notified of failure then a future update can add a write-property failed event.
//
// If no specific handler is set for the property then the default handler with name "" is invoked.
func (eThing *ExposedThing) handlePropertyWriteRequest(propName string, message []byte) {
	var err error
	logrus.Infof("Thing '%s'. property '%s'", eThing.TD.ID, propName)
	var propValue interface{}

	eThing.handlerMutex.RLock()
	defer eThing.handlerMutex.RUnlock()

	if eThing.propertyWriteHandlers == nil {
		logrus.Errorf("Handling property write request on a destroyed Exposed Thing")
		return
	}

	err = json.Unmarshal(message, &propValue)
	if err != nil {
		logrus.Warningf("Missing property value for %s: %s", propName, err)
		// TBD: reply with a failed event
		return
	}

	propAffordance := eThing.TD.GetProperty(propName)
	if propAffordance == nil {
		logrus.Warningf("property '%s' is not a valid name. Write request is ignored.", propName)
	} else if propAffordance.ReadOnly {
		logrus.Warningf("property '%s' is read-only. Write request is ignored.", propName)
	} else {
		propValue := thing.NewInteractionOutput(propValue, &propAffordance.DataSchema)
		// property specific handler takes precedence
		handler, _ := eThing.propertyWriteHandlers[propName]
		if handler != nil {
			err = handler(eThing, propName, propValue)
		} else {
			// default handler is a fallback
			defaultHandler, _ := eThing.propertyWriteHandlers[""]
			if defaultHandler == nil {
				logrus.Warningf("property '%s' has no write handler. Write request is ignored.", propName)
			} else {
				err = defaultHandler(eThing, propName, propValue)
			}
		}
	}

}

// SetActionHandler sets the handler for handling an action for the IoT device.
//  Only a single handler is active. If a handler is set when a previous handler was already set then the
//  latest handler will be used.
//
// The device code should implement this handler to updated configuration of the device.
//
// actionName is the action name this handler is for. If a single handler can take care of most actions
//  then use "" as the name to indicate it is the default handler.
//
// The handler should return nil if the write is accepted or an error if not accepted. The property value
// in the TD will be updated after the property has changed through the change notification handler.
func (eThing *ExposedThing) SetActionHandler(actionName string,
	actionHandler func(eThing *ExposedThing, actionName string, value *thing.InteractionOutput) error) {

	eThing.handlerMutex.Lock()
	defer eThing.handlerMutex.Unlock()
	eThing.actionHandlers[actionName] = actionHandler
}

// SetPropertyWriteHandler sets the handler for writing a property of the IoT device.
// This is intended to update device configuration. If the property is read-only the handler must return an error.
// Only a single handler is active. If a handler is set when a previous handler was already
//  set then the latest handler will be used.
//
// The device code should implement this handler to updated configuration of the device.
//
// propName is the property name this handler is for. Use "" for a default handler
//
// The handler should return nil if the request is accepted or an error if not accepted. The property value
// in the TD will be updated after the property has changed through the change notification handler.
func (eThing *ExposedThing) SetPropertyWriteHandler(propName string,
	writeHandler func(eThing *ExposedThing, propName string, value *thing.InteractionOutput) error) {

	eThing.handlerMutex.Lock()
	defer eThing.handlerMutex.Unlock()
	eThing.propertyWriteHandlers[propName] = writeHandler
}

// CreateExposedThing constructs an exposed thing from a TD.
//
// An exposed Thing is a local instance of a thing for the purpose of interaction with remote consumers.
// This is intended for use by the ExposedThingFactory only.
// Call 'Expose' to publish the TD of the thing and to start listening for actions and property write requests.
//
// * td is a Thing Description document of the Thing to expose.
// * mqttClient client for binding to the MQTT protocol
func CreateExposedThing(deviceID string, td *thing.ThingTD) *ExposedThing {
	eThing := &ExposedThing{
		DeviceID:              deviceID,
		TD:                    td,
		actionHandlers:        make(map[string]func(eThing *ExposedThing, actionName string, value *thing.InteractionOutput) error),
		propertyWriteHandlers: make(map[string]func(eThing *ExposedThing, actionName string, value *thing.InteractionOutput) error),
		valueStore:            make(map[string]*thing.InteractionOutput),
		valueStoreMutex:       sync.RWMutex{},
	}
	return eThing
}
