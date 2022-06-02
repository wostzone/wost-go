package consumedthing

import (
	"errors"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/wostzone/wost-go/pkg/mqttclient"
	"github.com/wostzone/wost-go/pkg/thing"
	"github.com/wostzone/wost-go/pkg/tlsclient"
)

// ConsumedThingProtocolBinding is the protocol binding for consumed things
//
// This:
//  1. Subscribes to events over MQTT events and updates Consumed Thing values
//  2. Handles requests to read properties from the directory service.
//  3. Installs the action hook to submit actions to exposed things over MQTT.
type ConsumedThingProtocolBinding struct {
	mqttClient *mqttclient.MqttClient
	authClient *tlsclient.TLSClient
	dirClient  *tlsclient.TLSClient
	td         *thing.ThingTD
	cThing     *ConsumedThing
}

// Handle incoming events or property update message.
//
// If the event is a property name then the payload is the property value according to the property affordance.
// If the event is an event name then the payload is the event value according to the event affordance.
// If the name is both an event and property defined in the TD then it is handled as both event and property.
//
//  address is the MQTT topic that the event is published on as: things/{thingID}/event/{eventName}
//  whereas message is the body of the event.
func (binding *ConsumedThingProtocolBinding) handleEvent(topic string, message []byte) {
	logrus.Infof("HandleEvent: received event on topic %s", topic)

	// the event topic is "things/id/event/name"
	parts := strings.Split(topic, "/")
	if len(parts) < 4 {
		logrus.Warningf("HandleEvent: EventName is missing in topic %s", topic)
		return
	}
	eventName := parts[3]
	_, found := binding.td.Events[eventName]
	if found {
		binding.cThing.HandleEvent(eventName, message)
	}
	_, found = binding.td.Properties[eventName]
	if found {
		binding.cThing.HandlePropertyChange(eventName, message)
	}
}

// InvokeAction publishes the action request
//
// @param cThing is the consumed thing invoking the action
// @param actionName name of the action to invoke as described in the TD actions section
// @param data parameters to pass to the action as defined in the TD schema
// Returns nil if the request is sent or an error if failed.
func (binding *ConsumedThingProtocolBinding) InvokeAction(actionName string, data interface{}) error {
	var err error
	action := binding.td.GetAction(actionName)
	if action == nil {
		err := errors.New("can't invoke action '" + actionName +
			"'. Action is not defined in TD '" + binding.td.ID + "'")
		logrus.Error(err)
	} else {
		topic := strings.ReplaceAll(TopicInvokeAction, "{thingID}", binding.td.ID) + "/" + actionName
		err = binding.mqttClient.PublishObject(topic, data)
		// TODO: reauthenticate if unauthorized
	}
	return err
}

//// ReadProperties requests a refresh of the cached property values of the thing
//// Properties will be refreshed in the background.
////
//// Returns nil if the request is sent or an error if failed.
//func (mb *ConsumedThingProtocolBinding) ReadProperties() error {
//	return nil
//}

// Start subscribes to Thing events
func (binding *ConsumedThingProtocolBinding) Start(
	authClient *tlsclient.TLSClient,
	dirClient *tlsclient.TLSClient,
	mqttClient *mqttclient.MqttClient,
) {
	binding.authClient = authClient
	binding.dirClient = dirClient
	binding.mqttClient = mqttClient
	// subscribe to all event messages of this thing
	topic := strings.ReplaceAll(TopicEmitEvent, "{thingID}", binding.td.ID) + "/#"
	binding.mqttClient.Subscribe(topic, binding.handleEvent)
}

// Stop unsubscribes from all messages
func (binding *ConsumedThingProtocolBinding) Stop() {
	topic := strings.ReplaceAll(TopicEmitEvent, "{thingID}", binding.td.ID) + "/#"
	binding.mqttClient.Unsubscribe(topic)
}

// WriteProperty publishes a request to change a property value in the exposed thing
// This does not update the property immediately. It is up to the exposedThing to perform necessary validation
// and notify subscribers with an property update event after the change has been applied.
//
// TODO: Currently if the message bus or the exposed thing refuses the request there is no reject message.
// The intent is to add support of a reject message when the MQTT broker rejects it, or
// when the exposed thing is offline or rejects the request due to an invalid payload.
//
// @param propName with the name of the property to write as defined in the Thing's TD document
// @param propValue with the new value
// Returns nil if the request is sent or an error if failed.
func (binding *ConsumedThingProtocolBinding) WriteProperty(propName string, propValue any) error {
	var err error
	topic := strings.ReplaceAll(TopicInvokeAction, "{thingID}", binding.td.ID) + "/" + propName
	err = binding.mqttClient.PublishObject(topic, propValue)
	return err
}

// CreateConsumedThingProtocolBinding creates the protocol binding for
// the consumed thing.
// Use 'Start' to subscribe and Stop to unsubscribe.
func CreateConsumedThingProtocolBinding(cThing *ConsumedThing) *ConsumedThingProtocolBinding {
	binding := &ConsumedThingProtocolBinding{
		cThing: cThing,
		td:     cThing.TD,
	}
	cThing.InvokeActionHook = binding.InvokeAction
	cThing.WritePropertyHook = binding.WriteProperty
	return binding
}
