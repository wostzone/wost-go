package exposedthing_test

import (
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wostzone/wost-go/pkg/exposedthing"
	"github.com/wostzone/wost-go/pkg/thing"
	"github.com/wostzone/wost-go/pkg/vocab"
)

func TestCreateExposedThing(t *testing.T) {
	logrus.Infof("--- TestCreateExposedThing ---")
	thingID := thing.CreateThingID("", testDeviceID, testDeviceType)
	td := createTestTD()
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)
	require.NotNil(t, eThing)
	assert.Equal(t, thingID, eThing.GetThingDescription().ID)

	eThing.Destroy()
}

func TestEmitEvent(t *testing.T) {
	logrus.Infof("--- TestEmitEvent ---")
	var rxEventName string
	var rxEventValue string

	// step 1 setup
	td := createTestTD()
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)
	props := make(map[string]interface{})
	props[testProp1Name] = testProp1Value
	eThing.EmitEventHook = func(name string, data interface{}) error {
		rxEventName = name
		rxEventValue = data.(string)
		return nil
	}
	// step 2 emit the event
	err := eThing.EmitEvent(testEventName, testProp1Value)
	assert.NoError(t, err)

	// validate
	assert.Equal(t, testEventName, rxEventName)
	assert.Equal(t, testProp1Value, rxEventValue)

	// cleanup
	eThing.Destroy()
}

func TestEmitEventNoHook(t *testing.T) {
	logrus.Infof("--- TestEmitEventNoHook ---")
	td := createTestTD()
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)
	// step 2 emit the event
	err := eThing.EmitEvent(testEventName, testProp1Value)
	assert.Error(t, err)
}

func TestEmitUnknownEventFails(t *testing.T) {
	logrus.Infof("--- TestEmitUnknownEventFails ---")
	td := createTestTD()
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)
	// step 2 emit the event
	err := eThing.EmitEvent("unknownEvent", testProp1Value)
	assert.Error(t, err)
}

func TestEmitPropertyChange(t *testing.T) {
	logrus.Infof("--- TestEmitPropertyChange ---")
	var rxPropValue = ""

	// step 1 setup
	td := createTestTD()
	props := make(map[string]interface{})
	props[testProp1Name] = testProp1Value
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)
	eThing.EmitPropertyChangeHook = func(prop string, data interface{}) error {
		rxPropValue = data.(string)
		return nil
	}
	// step 2 emit the property
	err := eThing.EmitPropertyChange(testProp1Name, testProp1Value, false)
	assert.NoError(t, err)

	// validate
	assert.Equal(t, testProp1Value, rxPropValue)

	// cleanup
	eThing.Destroy()
}

func TestHandleActionRequest(t *testing.T) {
	logrus.Infof("--- TestHandleActionRequest ---")
	var rxActionName string
	var rxActionValue string
	var rxDefaultName string
	var rxDefaultValue string
	var testValue1 = "value 1"

	// step 1 setup
	td := createTestTD()
	td.AddAction("action2", "test action", vocab.WoTDataTypeString)
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)
	eThing.SetActionHandler("",
		func(eThing *exposedthing.ExposedThing, name string, val *thing.InteractionOutput) error {
			rxDefaultName = name
			rxDefaultValue = val.ValueAsString()
			return nil
		})
	eThing.SetActionHandler(testActionName,
		func(eThing *exposedthing.ExposedThing, name string, val *thing.InteractionOutput) error {
			rxActionName = name
			rxActionValue = val.ValueAsString()
			return nil
		})

	// step 2 invoke action
	jsonValue, _ := json.Marshal(testValue1)
	eThing.HandleActionRequest(testActionName, jsonValue)
	eThing.HandleActionRequest("action2", jsonValue)

	assert.Equal(t, testActionName, rxActionName)
	assert.Equal(t, testValue1, rxActionValue)
	assert.Equal(t, "action2", rxDefaultName)
	assert.Equal(t, testValue1, rxDefaultValue)

	// step 3 cleanup
	eThing.Destroy()
}

func TestHandleActionRequestNoHandler(t *testing.T) {
	logrus.Infof("--- TestHandleActionRequestNoHandler ---")

	// step 1 setup
	td := createTestTD()
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)

	// step 2 invoke action
	jsonValue, _ := json.Marshal("somevalue")
	eThing.HandleActionRequest(testActionName, jsonValue)

	// check logging for an error
}

func TestHandlePropertyWriteRequest(t *testing.T) {
	logrus.Infof("--- TestHandlePropertyWriteRequest ---")
	var rxDefaultPropName string
	var rxDefaultPropValue string
	var rxPropName string
	var rxPropValue string
	var testValue1 = "value 1"

	// step 1 setup
	td := createTestTD()
	p2 := td.AddProperty("prop2", "test property", vocab.WoTDataTypeString)
	p2.ReadOnly = false
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)
	eThing.SetPropertyWriteHandler(testProp1Name,
		func(eThing *exposedthing.ExposedThing, name string, val *thing.InteractionOutput) error {
			rxPropName = name
			rxPropValue = val.ValueAsString()
			return nil
		})
	// default handler
	eThing.SetPropertyWriteHandler("",
		func(eThing *exposedthing.ExposedThing, name string, val *thing.InteractionOutput) error {
			rxDefaultPropName = name
			rxDefaultPropValue = val.ValueAsString()
			return nil
		})

	// step 2 invoke property write using actions
	jsonValue, _ := json.Marshal(testValue1)
	eThing.HandleActionRequest(testProp1Name, jsonValue)
	eThing.HandleActionRequest("prop2", jsonValue)

	assert.Equal(t, testProp1Name, rxPropName)
	assert.Equal(t, testValue1, rxPropValue)
	assert.Equal(t, "prop2", rxDefaultPropName)
	assert.Equal(t, testValue1, rxDefaultPropValue)

	// step 3 cleanup
	eThing.Destroy()
}

func TestHandlePropertyWriteRequestDestroyedThing(t *testing.T) {
	logrus.Infof("--- TestHandlePropertyWriteRequestDestroyedThing ---")

	// step 1 setup
	td := createTestTD()
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)
	eThing.SetPropertyWriteHandler("",
		func(eThing *exposedthing.ExposedThing, name string, val *thing.InteractionOutput) error {
			assert.Fail(t, "Should not write on a destroyed thing")
			return nil
		})

	// step 2 invoke property write
	eThing.Destroy()
	jsonValue, _ := json.Marshal(testProp1Value)
	eThing.HandleActionRequest(testProp1Name, jsonValue)

	// check log for error
}

func TestHandlePropertyWriteRequestNullData(t *testing.T) {
	logrus.Infof("--- TestHandlePropertyWriteRequestNullData ---")

	// step 1 setup
	td := createTestTD()
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)

	// step 2 invoke property write with nil data
	eThing.HandleActionRequest(testProp1Name, nil)

	// check log for warning
}

func TestHandlePropertyWriteRequestInvalidName(t *testing.T) {
	logrus.Infof("--- TestHandlePropertyWriteRequestInvalidName ---")

	// step 1 setup
	td := createTestTD()
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)
	eThing.SetPropertyWriteHandler("",
		func(eThing *exposedthing.ExposedThing, name string, val *thing.InteractionOutput) error {
			assert.Fail(t, "Should not write on an unknown prop")
			return nil
		})

	// step 2 invoke property write using actions
	jsonValue, _ := json.Marshal(testProp1Value)
	eThing.HandleActionRequest("prop2", jsonValue)
}

func TestHandlePropertyWriteRequestReadOnly(t *testing.T) {
	logrus.Infof("--- TestHandlePropertyWriteRequestReadOnly ---")

	// step 1 setup
	td := createTestTD()
	td.AddProperty("readonlyprop", "test readonly", vocab.WoTDataTypeString)
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)
	eThing.SetPropertyWriteHandler("",
		func(eThing *exposedthing.ExposedThing, name string, val *thing.InteractionOutput) error {
			assert.Fail(t, "Should not write on a readonly prop")
			return nil
		})

	// step 2 invoke property write using actions
	jsonValue, _ := json.Marshal(testProp1Value)
	eThing.HandleActionRequest("readonlyprop", jsonValue)
}

func TestHandlePropertyWriteNoHandler(t *testing.T) {
	logrus.Infof("--- TestHandlePropertyWriteNoHandler ---")

	// step 1 setup
	td := createTestTD()
	eThing := exposedthing.CreateExposedThing(testDeviceID, td)

	// step 2 invoke property write using actions
	jsonValue, _ := json.Marshal(testProp1Value)
	eThing.HandleActionRequest(testProp1Name, jsonValue)

	// no way to test what happened
}
