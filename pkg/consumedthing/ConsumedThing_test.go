package consumedthing_test

import (
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wostzone/wost-go/pkg/consumedthing"
	"github.com/wostzone/wost-go/pkg/thing"
	"github.com/wostzone/wost-go/pkg/vocab"
)

func TestCreateConsumedThing(t *testing.T) {
	logrus.Infof("--- TestCreateConsumedThing ---")
	thingID := thing.CreateThingID("", testDeviceID, testDeviceType)
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)
	require.NotNil(t, cThing)
	assert.Equal(t, thingID, cThing.GetThingDescription().ID)

	cThing.Stop()
}

func TestReceiveTD(t *testing.T) {
	logrus.Infof("--- TestReceiveTD ---")
	// step 1 setup
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)
	//assert.FailNow(t, "test not implemented")
	// step 5 cleanup
	cThing.Stop()
}
func TestSubscribeEvent(t *testing.T) {
	logrus.Infof("--- TestSubscribeEvent ---")
	const eventValue = "event1value"
	var eventCount = 0

	// step 1 setup
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)
	err := cThing.SubscribeEvent(testEventName,
		func(evName string, data *thing.InteractionOutput) {
			eventCount++
			assert.Equal(t, eventValue, data.ValueAsString())
		})
	assert.NoError(t, err)

	// step 2 pass the event value (impersonate a binding)
	jsonValue, _ := json.Marshal(eventValue)
	cThing.HandleEvent(testEventName, jsonValue)

	// step 3 the subscriber must have been notified
	assert.Equal(t, 1, eventCount)

	// step 4 cleanup
	cThing.Stop()
}

func TestSubscribeEventTwice(t *testing.T) {
	logrus.Infof("--- TestSubscribeEventTwice ---")

	// step 1 setup
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)
	err := cThing.SubscribeEvent(testEventName,
		func(evName string, data *thing.InteractionOutput) {
		})
	assert.NoError(t, err)

	// step 2 subscribing again should result in an error
	err = cThing.SubscribeEvent(testEventName,
		func(evName string, data *thing.InteractionOutput) {
		})
	assert.Error(t, err)
	cThing.Stop()
}

func TestObserveProperty(t *testing.T) {
	logrus.Infof("--- TestObserveProperty ---")
	var counter int32 = 0
	var issuedValue = 42
	var observedValue = 0

	// step 1 setup
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)

	err := cThing.ObserveProperty(testProp1Name,
		func(name string, data *thing.InteractionOutput) {
			assert.Equal(t, testProp1Name, name)
			atomic.AddInt32(&counter, 1)
			observedValue = data.ValueAsInt()
		})
	assert.NoError(t, err)

	// step 2 pass the property value in an event (impersonate a binding)
	jsonValue, _ := json.Marshal(issuedValue)
	cThing.HandlePropertyChange(testProp1Name, jsonValue)

	// step 3 observeProperty should have been invoked and match
	assert.NoError(t, err)
	assert.NotNil(t, observedValue)
	assert.Equal(t, issuedValue, observedValue)

	// step 4 read the property value. It should match
	val1, err := cThing.ReadProperty(testProp1Name)
	assert.Equal(t, issuedValue, val1.ValueAsInt())

	propNames := []string{testProp1Name}
	propInfo := cThing.ReadMultipleProperties(propNames)
	assert.Equal(t, len(propInfo), 1)
	assert.Equal(t, issuedValue, propInfo[testProp1Name].ValueAsInt())

	propInfo = cThing.ReadAllProperties()
	assert.GreaterOrEqual(t, len(propInfo), 1)

	// step 5 cleanup
	cThing.Stop()
}

// test with handling multiple properties
//func TestObserveProperties(t *testing.T) {
//	logrus.Infof("--- TestObserveProperties ---")
//	const testProp2Name = "prop2"
//	const value2 = "value2"
//	var counter int32 = 0
//
//	// step 1 setup
//	td := createTestTD()
//	td.AddProperty(testProp2Name, "test prop", vocab.WoTDataTypeString)
//	cThing := consumedthing.CreateConsumedThing(td)
//
//	err := cThing.ObserveProperty(testProp1Name,
//		func(name string, data *thing.InteractionOutput) {
//			assert.Equal(t, testProp1Name, name)
//			atomic.AddInt32(&counter, 1)
//		})
//	assert.NoError(t, err)
//
//	// step 2 create multiple properties
//	props := make(map[string]interface{})
//	props[testProp1Name] = testProp1Value
//	props[testProp2Name] = value2
//	jsonValue, _ := json.Marshal(props)
//	cThing.HandlePropertyChange(consumedthing.TopicSubjectProperties, jsonValue)
//
//	// step 3 both should to be received
//	assert.Equal(t, int32(1), counter)
//	io2, err := cThing.ReadProperty(testProp2Name)
//	assert.NoError(t, err)
//	assert.Equal(t, value2, io2.ValueAsString())
//
//	// step 4 cleanup
//	cThing.Stop()
//}

// test with handling property that isn't a map
func TestObservePropertyNotAMap(t *testing.T) {
	logrus.Infof("--- TestObservePropertyNotAMap ---")
	const testProp2Name = "prop2"
	const value2 = "value2"
	var counter int32 = 0

	// step 1 setup
	td := createTestTD()
	td.AddProperty(testProp2Name, "test prop", vocab.WoTDataTypeString)
	cThing := consumedthing.CreateConsumedThing(td)

	err := cThing.ObserveProperty(testProp1Name,
		func(name string, data *thing.InteractionOutput) {
			assert.Equal(t, testProp1Name, name)
			atomic.AddInt32(&counter, 1)
		})
	assert.NoError(t, err)

	jsonValue, _ := json.Marshal(value2)
	cThing.HandleEvent(consumedthing.TopicSubjectProperties, jsonValue)
	assert.Equal(t, int32(0), counter)
}

// test with handling property that isn't in the TD
func TestObservePropertyNotInTD(t *testing.T) {
	logrus.Infof("--- TestObservePropertyNotInTD ---")
	const testProp3Name = "unknownProp"
	const value3 = "value3"

	// step 1 setup
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)

	err := cThing.ObserveProperty(testProp3Name,
		func(name string, data *thing.InteractionOutput) {
			assert.Fail(t, "Received property notification but prop is not in TD")
		})
	assert.NoError(t, err)

	props := make(map[string]interface{})
	props[testProp3Name] = value3
	jsonValue, _ := json.Marshal(props)
	cThing.HandleEvent(consumedthing.TopicSubjectProperties, jsonValue)
}

func TestObservePropertyTwiceShouldFail(t *testing.T) {
	logrus.Infof("--- TestObservePropertyTwiceShouldFail ---")

	// step 1 setup
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)
	err := cThing.ObserveProperty(testProp1Name,
		func(evName string, data *thing.InteractionOutput) {
		})
	assert.NoError(t, err)

	// step 2 subscribing again should result in an error
	err = cThing.ObserveProperty(testProp1Name,
		func(evName string, data *thing.InteractionOutput) {
		})
	assert.Error(t, err)
}

func TestWriteProperties(t *testing.T) {
	logrus.Infof("--- TestWriteProperties ---")
	var receivedPropName string
	var receivedPropValue string

	// step 1 setup
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)
	cThing.WritePropertyHook = func(propName string, params interface{}) error {
		receivedPropName = propName
		receivedPropValue = params.(string)
		return nil
	}

	// step 3 submit the write request
	props := make(map[string]interface{})
	props[testProp1Name] = testProp1Value
	err := cThing.WriteMultipleProperties(props)
	assert.NoError(t, err)
	assert.Equal(t, testProp1Name, receivedPropName)
	assert.Equal(t, testProp1Value, receivedPropValue)

	// step 4 cleanup
	cThing.Stop()
}

func TestWritePropertiesNoHook(t *testing.T) {
	logrus.Infof("--- TestWritePropertiesNoHook ---")

	// step 1 setup
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)
	// step 2 - no hook is installed

	// step 3 submit the write request and expect an error
	props := make(map[string]interface{})
	props[testProp1Name] = testProp1Value
	err := cThing.WriteMultipleProperties(props)
	assert.Error(t, err)
}

func TestInvokeAction(t *testing.T) {
	logrus.Infof("--- TestInvokeAction ---")
	var receivedActionName string
	var receivedActionValue string

	// step 1 setup
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)
	cThing.InvokeActionHook = func(name string, params interface{}) error {
		receivedActionName = name
		receivedActionValue = params.(string)
		return nil
	}
	err := cThing.InvokeAction(testActionName, "bob")
	assert.NoError(t, err)
	assert.Equal(t, testActionName, receivedActionName)
	assert.Equal(t, "bob", receivedActionValue)

	err = cThing.InvokeAction("badname", "bob")
	assert.Error(t, err)
}

func TestInvokeActionNoHook(t *testing.T) {
	logrus.Infof("--- TestInvokeActionNoHook ---")

	// step 1 setup
	td := createTestTD()
	cThing := consumedthing.CreateConsumedThing(td)

	err := cThing.InvokeAction(testActionName, "bob")
	assert.Error(t, err)
}
