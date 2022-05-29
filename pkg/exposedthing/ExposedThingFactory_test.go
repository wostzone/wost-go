// Package exposedthing_test with tests for the factory and binding
package exposedthing_test

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wostzone/wost-go/pkg/accounts"
	"github.com/wostzone/wost-go/pkg/consumedthing"
	"github.com/wostzone/wost-go/pkg/exposedthing"
	"github.com/wostzone/wost-go/pkg/testenv"
	"github.com/wostzone/wost-go/pkg/thing"
	"net/http"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

var testCerts = testenv.CreateCertBundle()
var testServer *http.Server
var testMosqCmd *exec.Cmd
var tempFolder string // for test environment

func setupTestFactory(connect bool) (*exposedthing.ExposedThingFactory, error) {
	var err error
	tempFolder = path.Join(os.TempDir(), "wost-test-exposedthing")
	testServer = testenv.StartServices(&testCerts)
	testMosqCmd, err = testenv.StartMosquitto(&testCerts, tempFolder)
	time.Sleep(time.Second)

	factory := exposedthing.CreateExposedThingFactory(testAppID, testCerts.PluginCert, testCerts.CaCert)
	if connect {
		err = factory.Connect(testenv.ServerAddress, testenv.MqttPortCert)
	}
	return factory, err
}

func tearDown(factory *exposedthing.ExposedThingFactory) {
	factory.Disconnect()
	_ = testMosqCmd.Process.Kill()
	_ = testServer.Close()
	testenv.StopMosquitto(testMosqCmd, tempFolder)
}

func TestCreateFactory(t *testing.T) {
	logrus.Infof("--- TestCreateFactory ---")

	factory, err := setupTestFactory(true)
	assert.NoError(t, err)
	require.NotNil(t, factory)
	tearDown(factory)
}

func TestExposeDestroyThing(t *testing.T) {
	logrus.Infof("--- TestExposeDestroyThing ---")

	factory, _ := setupTestFactory(true)
	td := createTestTD()
	eThing := factory.Expose(testDeviceID, td)
	assert.NotNil(t, eThing)
	factory.Destroy(eThing)

	tearDown(factory)
}

func TestExposedThing_EmitEvent(t *testing.T) {
	const value1 = "value1"
	logrus.Infof("--- TestExposedThing_EmitEvent ---")

	factory, _ := setupTestFactory(true)
	td := createTestTD()
	eThing := factory.Expose(testDeviceID, td)
	assert.NotNil(t, eThing)

	err := eThing.EmitEvent(testEventName, value1)
	assert.NoError(t, err)

	factory.Destroy(eThing)
	tearDown(factory)
}

func TestExposedThing_EmitPropertiesChange(t *testing.T) {
	logrus.Infof("--- TestExposedThing_EmitPropertiesChange ---")

	factory, _ := setupTestFactory(true)
	td := createTestTD()
	eThing := factory.Expose(testDeviceID, td)
	assert.NotNil(t, eThing)

	err := eThing.EmitPropertyChange(testProp1Name, testProp1Value)
	assert.NoError(t, err)

	factory.Destroy(eThing)
	tearDown(factory)
}

func TestEmitUnknownPropertyChange(t *testing.T) {
	logrus.Infof("--- TestEmitUnknownPropertyChange ---")

	factory, _ := setupTestFactory(true)
	td := createTestTD()
	eThing := factory.Expose(testDeviceID, td)
	assert.NotNil(t, eThing)

	err := eThing.EmitPropertyChange(testProp1Name, "value")
	assert.NoError(t, err)

	err = eThing.EmitPropertyChange("notaproperty", "value")
	assert.Error(t, err)

	factory.Destroy(eThing)
	tearDown(factory)
}

func TestEmitPropertyChangeNotConnected(t *testing.T) {
	logrus.Infof("--- TestEmitUnknownPropertyChange ---")

	factory, _ := setupTestFactory(true)
	td := createTestTD()
	eThing := factory.Expose(testDeviceID, td)
	assert.NotNil(t, eThing)

	factory.Disconnect()
	err := eThing.EmitPropertyChange(testProp1Name, "value")
	assert.Error(t, err)

	tearDown(factory)
}

func TestExposedThing_HandleActionRequest(t *testing.T) {
	const value1 = "value1"
	var rxValue string
	logrus.Infof("--- TestExposedThing_HandleActionRequest ---")

	// step 1: create the exposed thing from the test TD
	factory, _ := setupTestFactory(true)
	td := createTestTD()
	eThing := factory.Expose(testDeviceID, td)
	assert.NotNil(t, eThing)
	eThing.SetActionHandler(testActionName,
		func(eThing *exposedthing.ExposedThing, actionName string, value *thing.InteractionOutput) error {
			assert.Equal(t, testActionName, actionName)
			rxValue = value.ValueAsString()
			return nil
		})

	// step 2: setup the consumed side to emit an action
	account := accounts.AccountRecord{
		Address:   testenv.ServerAddress,
		MqttPort:  testenv.MqttPortCert,
		LoginName: "sss",
		Enabled:   true,
	}
	cFactory := consumedthing.CreateConsumedThingFactory(
		"etTest", &account, testCerts.CaCert)
	err := cFactory.ConnectWithCert(testCerts.PluginCert)
	require.NoError(t, err)
	cThing := cFactory.Consume(td)

	// step 3 run the test and check result
	err = cThing.InvokeAction(testActionName, value1)
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 100)

	// cleanup and test
	factory.Destroy(eThing)
	tearDown(factory)

	assert.Equal(t, value1, rxValue)
}

func TestExposedThing_HandleWritePropertyRequest(t *testing.T) {
	const value2 = "value2"
	var rxValue string
	logrus.Infof("--- TestExposedThing_HandleWritePropertyRequest ---")

	// step 1: create the exposed thing from the test TD
	factory, _ := setupTestFactory(true)
	td := createTestTD()
	eThing := factory.Expose(testDeviceID, td)
	assert.NotNil(t, eThing)
	eThing.SetPropertyWriteHandler("",
		func(eThing *exposedthing.ExposedThing, propName string, value *thing.InteractionOutput) error {
			assert.Equal(t, testProp1Name, propName)
			rxValue = value.ValueAsString()
			return nil
		})

	// step 2: setup the consumed side to write a property
	account := accounts.AccountRecord{
		Address:   testenv.ServerAddress,
		MqttPort:  testenv.MqttPortCert,
		LoginName: "sss",
		Enabled:   true,
	}
	cFactory := consumedthing.CreateConsumedThingFactory(
		"etTest", &account, testCerts.CaCert)
	err := cFactory.ConnectWithCert(testCerts.PluginCert)
	require.NoError(t, err)
	cThing := cFactory.Consume(td)

	// step 3 run the test and check result
	err = cThing.WriteProperty(testProp1Name, value2)
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 100)

	// cleanup and test
	factory.Destroy(eThing)
	tearDown(factory)

	assert.Equal(t, value2, rxValue)
}

//
//func TestHandleActionRequest(t *testing.T) {
//	logrus.Infof("--- TestHandleActionRequest ---")
//	var receivedActionDefaultHandler bool = false
//	var receivedActionHandler bool = false
//	var rxMutex = sync.RWMutex{}
//
//	// step 1 create the MQTT message bus client
//	client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
//	err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
//	assert.NoError(t, err)
//
//	// step 2 create a ConsumedThing and ExposedThing with handlers
//	testTD.UpdateAction("action2", &thing.ActionAffordance{})
//	cThing := mqttbinding.Consume(testTD, client)
//	eThing := mqttbinding.CreateExposedThing(testDeviceID, testTD, client)
//	eThing.SetActionHandler("",
//		func(eThing *mqttbinding.MqttExposedThing, name string, val mqttbinding.InteractionOutput) error {
//			rxMutex.Lock()
//			defer rxMutex.Unlock()
//			receivedActionDefaultHandler = true
//			return nil
//		})
//	eThing.SetActionHandler(testActionName,
//		func(eThing *mqttbinding.MqttExposedThing, name string, val mqttbinding.InteractionOutput) error {
//			receivedActionHandler = testActionName == name
//			rxMutex.Lock()
//			defer rxMutex.Unlock()
//			return nil
//		})
//	err = eThing.Expose()
//	assert.NoError(t, err)
//	assert.NotNil(t, eThing)
//	time.Sleep(time.Second)
//
//	// step 3  an action
//	err = cThing.InvokeAction(testActionName, "hi there")
//	assert.NoError(t, err)
//	time.Sleep(time.Second)
//	rxMutex.RLock()
//	assert.True(t, receivedActionHandler)
//	assert.False(t, receivedActionDefaultHandler)
//	rxMutex.RUnlock()
//
//	// step 4 test result. Both exposed and consumed thing must have the new value
//	err = cThing.InvokeAction("action2", nil)
//	assert.NoError(t, err)
//	time.Sleep(time.Second)
//
//	// step 4 test result. Both exposed and consumed thing must have the new value
//	rxMutex.RLock()
//	assert.True(t, receivedActionDefaultHandler)
//	rxMutex.RUnlock()
//
//	// step 5 cleanup
//	eThing.Destroy()
//	client.Close()
//}
//
//func TestHandleActionRequestInvalidParams(t *testing.T) {
//	logrus.Infof("--- TestHandleActionRequestInvalidParams ---")
//	var receivedAction bool = false
//
//	// step 1 create the MQTT message bus client
//	client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
//	err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
//	assert.NoError(t, err)
//
//	// step 2 create a ConsumedThing and ExposedThing with default handler
//	eThing := mqttbinding.CreateExposedThing(testDeviceID, testTD, client)
//	eThing.SetActionHandler("",
//		func(eThing *mqttbinding.MqttExposedThing, name string, val mqttbinding.InteractionOutput) error {
//			receivedAction = true
//			return nil
//		})
//	err = eThing.Expose()
//	assert.NoError(t, err)
//	assert.NotNil(t, eThing)
//
//	// step 3  an action with no name
//	topic := strings.ReplaceAll(mqttbinding.TopicInvokeAction, "{thingID}", testThingID)
//	err = client.Publish(topic, []byte(testProp1Value))
//	assert.NoError(t, err)
//	time.Sleep(time.Second)
//
//	// step 4  an unregistered action
//	topic = strings.ReplaceAll(mqttbinding.TopicInvokeAction, "{thingID}", testThingID) + "/badaction"
//	err = client.Publish(topic, []byte(testProp1Value))
//	assert.NoError(t, err)
//	time.Sleep(time.Second)
//
//	// step 5 test result. no action should have been triggered
//	assert.False(t, receivedAction)
//
//	// step 5 cleanup
//	eThing.Destroy()
//	client.Close()
//}
//
//func TestHandleActionRequestNoHandler(t *testing.T) {
//	logrus.Infof("--- TestHandleActionRequestNoHandler ---")
//
//	// step 1 create the MQTT message bus client
//	client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
//	err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
//	assert.NoError(t, err)
//
//	// step 2 create a ConsumedThing and ExposedThing with default handler
//	cThing := mqttbinding.Consume(testTD, client)
//	eThing := mqttbinding.CreateExposedThing(testDeviceID, testTD, client)
//	// no action handler
//	err = eThing.Expose()
//	assert.NoError(t, err)
//	assert.NotNil(t, cThing)
//	assert.NotNil(t, eThing)
//
//	// step 3 invoke action with no handler
//	err = cThing.InvokeAction(testActionName, "")
//	assert.NoError(t, err)
//	time.Sleep(time.Second)
//
//	// missing handler does not return an error, just an error in the log
//
//	// step 5 cleanup
//	eThing.Destroy()
//	client.Close()
//}
//
//func TestEmitEventNotConnected(t *testing.T) {
//	logrus.Infof("--- TestEmitEventNotConnected ---")
//
//	// step 1 create the MQTT message bus client but dont connect
//	client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
//	//err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
//	//assert.NoError(t, err)
//
//	// step 2 create an ExposedThing
//	eThing := mqttbinding.CreateExposedThing(testDeviceID, testTD, client)
//	err := eThing.Expose()
//	assert.Error(t, err, "Expect no connection error")
//
//	// step 3 emit event
//	err = eThing.EmitEvent(testEventName, "")
//	assert.Error(t, err)
//
//	// step 5 cleanup
//	eThing.Destroy()
//	client.Close()
//}
//
//func TestEmitEventNotFound(t *testing.T) {
//	logrus.Infof("--- TestEmitEventNotFound ---")
//
//	// step 1 create the MQTT message bus client and connect
//	client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
//	err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
//	assert.NoError(t, err)
//
//	// step 2 create an ExposedThing
//	eThing := mqttbinding.CreateExposedThing(testDeviceID, testTD, client)
//	err = eThing.Expose()
//	assert.NoError(t, err)
//
//	// step 3 emit unknown event
//	err = eThing.EmitEvent("unknown-event", "")
//	assert.Error(t, err)
//
//	// step 5 cleanup
//	eThing.Destroy()
//	client.Close()
//}
