package consumedthing

import (
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/wostzone/wost-go/pkg/mqttclient"
	"github.com/wostzone/wost-go/pkg/thing"
	"github.com/wostzone/wost-go/pkg/tlsclient"
	"strings"
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

// Handle incoming events.
// If the event name is 'properties' then the payload is a map of property name-value pairs.
// If the event is a propertyName then the payload is the property value of that event.
// Otherwise the event payload is described in the TD event affordance.
// Last invoke the subscriber to the event name, if any, or the default subscriber
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

	binding.cThing.HandleEvent(eventName, message)
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
