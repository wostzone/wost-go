// Package exposedthing that implements the ExposedThing MQTT protocol binding
package exposedthing

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/wostzone/wost-go/pkg/consumedthing"
	"github.com/wostzone/wost-go/pkg/mqttclient"
	"github.com/wostzone/wost-go/pkg/thing"
	"strings"
)

// ExposedThingMqttBinding that connects the exposed thing to the message bus
type ExposedThingMqttBinding struct {
	eThing     *ExposedThing
	mqttClient *mqttclient.MqttClient
	td         *thing.ThingTD
}

// EmitEvent publishes a single event to subscribers.
// The topic will be things/{thingID}/event/{name} and payload will be the event data.
// If the event cannot be published, for example it is not defined, an error is returned.
//
// name is the name of the event as described in the TD, or one of the general purpose events.
// data is the event value as defined in the TD events schema and used as the payload
// Returns an error if the event is not found or cannot be published
func (binding *ExposedThingMqttBinding) EmitEvent(name string, data interface{}) error {
	topic := strings.ReplaceAll(consumedthing.TopicEmitEvent, "{thingID}", binding.td.ID) + "/" + name
	err := binding.mqttClient.PublishObject(topic, data)
	return err
}

// EmitPropertiesChange sends a properties change event for multiple properties
// and if the property name matches an event name, an event with the property name
// is sent, if the value changed.
//
// This uses the 'TopicEmitPropertiesChange' topic, eg 'things/{thingID}/event/properties'.
// propMap is a map of property name to raw value. This will be converted to json as-is.
//
// Returns an error if submitting an event fails
func (binding *ExposedThingMqttBinding) EmitPropertiesChange(propMap map[string]interface{}) error {
	topic := strings.ReplaceAll(
		consumedthing.TopicEmitPropertiesChange, "{thingID}", binding.td.ID)
	err := binding.mqttClient.PublishObject(topic, propMap)
	if err != nil {
		logrus.Warningf("EmitPropertyChanges: Failed %s", err)
		return err
	}
	cpAsText, _ := json.Marshal(propMap)
	logrus.Infof("EmitPropertyChanges: submitted %d properties for thing %s: %s",
		len(propMap), binding.td.ID, cpAsText)
	return err
}

// Handle action requests for this Thing
// This passes the request to the registered handler
// If no specific handler is set then the default handler with name "" is invoked.
func (binding *ExposedThingMqttBinding) handleActionRequest(address string, message []byte) {
	logrus.Infof("ExposedThing.HandleActionRequest: address '%s', message: '%s'", address, message)

	// the topic is "things/id/action/actionName"
	thingID, messageType, actionName := consumedthing.SplitTopic(address)
	if thingID == "" || messageType == "" {
		logrus.Warningf("ExposedThing.HandleActionRequest: actionName is missing in topic %s", address)
		return
	}
	binding.eThing.HandleActionRequest(actionName, message)
}

// handlePropertyWriteRequest for updating a property
// This invokes the property update handler with the value of the new property.
//
// It is up to the handler to invoke emitPropertyChange and update the property in the valueStore
// after the change takes effect.
//
// There is currently no error feedback in case the request cannot be handled. The requester will receive a
// property change event when the request has completed successfully.
// Failure to complete the request can be caused by an invalid value or if the IoT device is not
// in a state to accept changes.
//
// TBD: if there is a need to be notified of failure then a future update can add a write-property failed event.
//
// If no specific handler is set for the property then the default handler with name "" is invoked.
//func (eThing *ExposedThing) handlePropertyWriteRequest(propName string, propAffordance *thing.PropertyAffordance, message []byte) {
//	var err error
//	logrus.Infof("ExposedThing.handlePropertyWriteRequest for '%s'. property '%s'", eThing.td.ID, propName)
//	var propValue interface{}
//
//	err = json.Unmarshal(message, &propValue)
//	if err != nil {
//		logrus.Warningf("ExposedThing.handlePropertyWriteRequest: missing property value for %s: %s", propName, err)
//		// TBD: reply with a failed event
//		return
//	}
//
//	if propAffordance == nil {
//		err = errors.New("property '%s' is not a valid name")
//		logrus.Warningf("ExposedThing.handlePropertyWriteRequest: %s. Request ignored.", err)
//	} else if propAffordance.ReadOnly {
//		err = errors.New("property '" + propName + "' is readonly")
//		logrus.Warningf("ExposedThing.handlePropertyWriteRequest: %s", err)
//	} else {
//		propValue := NewInteractionOutput(propValue, &propAffordance.DataSchema)
//		// property specific handler takes precedence
//		handler, _ := eThing.propertyWriteHandlers[propName]
//		if handler != nil {
//			err = handler(eThing, propName, propValue)
//		} else {
//			// default handler is a fallback
//			defaultHandler, _ := eThing.propertyWriteHandlers[""]
//			if defaultHandler == nil {
//				err = errors.New("no handler for property write request")
//				logrus.Warningf("ExposedThing.handlePropertyWriteRequest: No handler for property '%s' on thing '%s'", propName, eThing.td.ID)
//			} else {
//				err = defaultHandler(eThing, propName, propValue)
//			}
//		}
//	}
//
//}

//func (eThing *ExposedThing) SetPropertyReadHandler(func(name string) string) error {
//	return errors.New("not implemented")
//}

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
//func (eThing *ExposedThing) SetActionHandler(
//	actionName string, actionHandler func(eThing *ExposedThing, actionName string, value InteractionOutput) error) {
//
//	eThing.actionHandlers[actionName] = actionHandler
//}

// SetPropertyObserveHandler sets the handler for subscribing to properties
// Not implemented as subscriptions are handled by the MQTT message bus
//func (eThing *ExposedThing) SetPropertyObserveHandler(handler func(name string) InteractionOutput) error {
//	_ = handler
//	return errors.New("not implemented")
//}

// SetPropertyUnobserveHandler sets the handler for unsubscribing to properties
// Not implemented as subscriptions are handled by the MQTT message bus
//func (eThing *ExposedThing) SetPropertyUnobserveHandler(handler func(name string) InteractionOutput) error {
//	_ = handler
//	return errors.New("not implemented")
//}

// SetPropertyReadHandler sets the handler for reading a property of the IoT device
// Not implemented as property values are updated with events and not requested.
// The latest property value can be found with the TD properties.
//func (eThing *ExposedThing) SetPropertyReadHandler(handler func(name string) string) error {
//	_ = handler
//	return errors.New("not implemented")
//}

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
//func (eThing *ExposedThing) SetPropertyWriteHandler(
//	propName string,
//	writeHandler func(eThing *ExposedThing, propName string, value InteractionOutput) error) {
//
//	eThing.propertyWriteHandlers[propName] = writeHandler
//}

// Start subscribes to Thing action requests
func (binding *ExposedThingMqttBinding) Start() {
	// subscribe to action messages for the thing
	topic := strings.ReplaceAll(consumedthing.TopicInvokeAction, "{thingID}", binding.td.ID) + "/#"
	binding.mqttClient.Subscribe(topic, binding.handleActionRequest)
}

// Stop unsubscribes from all messages
func (binding *ExposedThingMqttBinding) Stop() {
	topic := strings.ReplaceAll(consumedthing.TopicInvokeAction, "{thingID}", binding.td.ID) + "/#"
	binding.mqttClient.Unsubscribe(topic)
}

// CreateExposedThingMqttBinding constructs a mqtt protocol binding for exposed things.
//
//
// @param eThing is the Exposed Thing to bind to
// mqttClient MQTT client for binding to the MQTT protocol
func CreateExposedThingMqttBinding(eThing *ExposedThing, mqttClient *mqttclient.MqttClient) *ExposedThingMqttBinding {
	binding := &ExposedThingMqttBinding{
		eThing:     eThing,
		mqttClient: mqttClient,
	}
	return binding
}
