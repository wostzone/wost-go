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
		logrus.Warningf("Failed %s", err)
		return err
	}
	cpAsText, _ := json.Marshal(propMap)
	logrus.Infof("Submitted %d properties for thing %s: %s",
		len(propMap), binding.td.ID, cpAsText)
	return err
}

// Handle action requests for this Thing.
//
// This passes the request to the registered handler.
// If no specific handler is set then the default handler with name "" is invoked.
//
// Since property write requests are sent as actions, this also handles these
// requests. In this case the action name is the property name.
func (binding *ExposedThingMqttBinding) handleActionRequest(address string, message []byte) {
	logrus.Infof("address '%s', message: '%s'", address, message)

	// the topic is "things/id/action/actionName"
	thingID, messageType, actionName := consumedthing.SplitTopic(address)
	if thingID == "" || messageType == "" {
		logrus.Warningf("actionName is missing in topic %s", address)
		return
	}
	binding.eThing.HandleActionRequest(actionName, message)
}

// Start subscribes to Thing action requests
// Publish the Thing's own TD
func (binding *ExposedThingMqttBinding) Start() {
	logrus.Infof("binding for exposed thing '%s'", binding.td.ID)
	// subscribe to action/property write messages for the thing
	topic := strings.ReplaceAll(consumedthing.TopicInvokeAction, "{thingID}", binding.td.ID) + "/#"
	binding.mqttClient.Subscribe(topic, binding.handleActionRequest)

	topic = strings.ReplaceAll(consumedthing.TopicThingTD, "{thingID}", binding.td.ID)
	err := binding.mqttClient.PublishObject(topic, binding.td)
	// TBD how to handle the error?
	_ = err
}

// Stop unsubscribes from all messages
func (binding *ExposedThingMqttBinding) Stop() {
	logrus.Infof("binding for exposed thing '%s'", binding.td.ID)
	topic := strings.ReplaceAll(consumedthing.TopicInvokeAction, "{thingID}", binding.td.ID) + "/#"
	binding.mqttClient.Unsubscribe(topic)
}

// CreateExposedThingMqttBinding constructs a mqtt protocol binding for exposed things.
//
//  eThing is the Exposed Thing to bind to
//  mqttClient MQTT client for binding to the MQTT protocol
func CreateExposedThingMqttBinding(eThing *ExposedThing, mqttClient *mqttclient.MqttClient) *ExposedThingMqttBinding {
	binding := &ExposedThingMqttBinding{
		td:         eThing.TD,
		eThing:     eThing,
		mqttClient: mqttClient,
	}
	eThing.EmitPropertiesChangeHook = binding.EmitPropertiesChange
	eThing.EmitEventHook = binding.EmitEvent
	return binding
}
