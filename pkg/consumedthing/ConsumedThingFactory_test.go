package consumedthing_test

import (
	"crypto/x509"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/wostzone/wost-go/pkg/accounts"
	"github.com/wostzone/wost-go/pkg/consumedthing"
	"github.com/wostzone/wost-go/pkg/testenv"
	"testing"
)

const testAuthPort = 9881

var testCACert *x509.Certificate

//// CA, server and plugin test certificate
//var certs testenv.TestCerts
//
//var homeFolder string
//var configFolder string
//
//// Use test/mosquitto-test.conf and a client cert port
//var mqttCertAddress = fmt.Sprintf("%s:%d", testenv.ServerAddress, testenv.MqttPortCert)
//
//var testThingID = thing.CreateThingID("", testDeviceID, testDeviceType)
//var testTD = createTestTD()
//
//// TestMain - launch mosquitto to publish and subscribe
//func TestMain(m *testing.M) {
//	cwd, _ := os.Getwd()
//	homeFolder = path.Join(cwd, "../../test")
//	configFolder = path.Join(homeFolder, "config")
//	certFolder := path.Join(homeFolder, "certs")
//	_ = os.Chdir(homeFolder)
//
//	logging.SetLogging("info", "")
//	certs = testenv.CreateCertBundle()
//	mosquittoCmd, err := testenv.StartMosquitto(configFolder, certFolder, &certs)
//	if err != nil {
//		logrus.Fatalf("Unable to start mosquitto: %s", err)
//	}
//
//	result := m.Run()
//	testenv.StopMosquitto(mosquittoCmd)
//	os.Exit(result)
//}

// Create a test server for authentication, directory and mqtt broker
// these services run on localhost
func createTestServices(certs testenv.TestCerts) {
	//testAuthService :=
}

// Create a test factory with auth and mqtt clients
// Use createTestServices to fake a server
func createTestFactory() *consumedthing.ConsumedThingFactory {
	certs := testenv.CreateCertBundle()
	testCACert = certs.CaCert

	account := &accounts.AccountRecord{
		Address:   testenv.ServerAddress,
		LoginName: "user1",
		MqttPort:  testenv.MqttPortUnpw,
		AuthPort:  testAuthPort,
	}
	factory := consumedthing.CreateConsumedThingFactory(testAppID, account, testCACert)
	return factory
}

func TestCreateFactory(t *testing.T) {
	logrus.Infof("--- TestStartStopFactory ---")

	factory := createTestFactory()
	factory.Disconnect()
}

func TestConsumeDestroyThing(t *testing.T) {
	logrus.Infof("--- TestDestroyThing ---")

	factory := createTestFactory()
	td := createTestTD()
	ct := factory.Consume(td)
	factory.Destroy(ct)
}

func TestConnect(t *testing.T) {
	logrus.Infof("--- TestConnect ---")

	factory := createTestFactory()
	td := createTestTD()
	store := factory.GetThingStore()
	store.AddTD(td)
	err := factory.Connect("")
	assert.Error(t, err)
}

//func TestReadProperty(t *testing.T) {
//	logrus.Infof("--- TestReadProperty ---")
//	var observedProperty int32 = 0
//
//	// step 1 create the MQTT message bus client
//	client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
//	err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
//	assert.NoError(t, err)
//
//	// step 2 create a ConsumedThing
//	cThing := mqttbinding.Consume(testTD, client)
//	err = cThing.ObserveProperty(testProp1Name, func(name string, data mqttbinding.InteractionOutput) {
//		assert.Equal(t, testProp1Name, name)
//		atomic.AddInt32(&observedProperty, 1)
//	})
//	assert.NoError(t, err)
//
//	// step 3 publish the property value (impersonate an ExposedThing)
//	//topic := strings.ReplaceAll(mqttbinding.TopicEmitEvent, "{thingID}", testThingID) + "/" + testProp1Name
//	topic := mqttbinding.CreateTopic(testThingID, mqttbinding.TopicTypeEvent) + "/" + testProp1Name
//	err = client.PublishObject(topic, testProp1Value)
//	assert.NoError(t, err)
//	time.Sleep(time.Second)
//
//	// step 4 read the property value. It should match
//	val1, err := cThing.ReadProperty(testProp1Name)
//	assert.NoError(t, err)
//	assert.NotNil(t, val1)
//	assert.Equal(t, int32(1), atomic.LoadInt32(&observedProperty))
//
//	propNames := []string{testProp1Name}
//	propInfo := cThing.ReadMultipleProperties(propNames)
//	assert.Equal(t, len(propInfo), 1)
//
//	propInfo = cThing.ReadAllProperties()
//	assert.GreaterOrEqual(t, len(propInfo), 1)
//
//	// step 5 cleanup
//	cThing.Stop()
//	client.Disconnect()
//}
//
//func TestReceiveEvent(t *testing.T) {
//	logrus.Infof("--- TestReceiveEvent ---")
//	const eventName = "event1"
//	const eventValue = "hello world"
//	var receivedEvent int32 = 0
//
//	// step 1 create the MQTT message bus client
//	client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
//	err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
//	assert.NoError(t, err)
//
//	// step 2 create a ConsumedThing and subscribe to event
//	cThing := mqttbinding.Consume(testTD, client)
//	err = cThing.SubscribeEvent(eventName, func(ev string, data mqttbinding.InteractionOutput) {
//		if eventName == ev {
//			atomic.AddInt32(&receivedEvent, 1)
//		}
//		receivedText := data.ValueAsString()
//		assert.Equal(t, eventValue, receivedText)
//	})
//	assert.NoError(t, err)
//
//	// step 3 publish the event (impersonate an ExposedThing)
//	topic := strings.ReplaceAll(mqttbinding.TopicEmitEvent, "{thingID}", testThingID) + "/" + eventName
//	err = client.PublishObject(topic, eventValue)
//	assert.NoError(t, err)
//	time.Sleep(time.Second)
//
//	// step 4 check result
//	assert.Equal(t, int32(1), atomic.LoadInt32(&receivedEvent))
//
//	// step 5 cleanup
//	cThing.Stop()
//	client.Disconnect()
//}
//
//func TestInvokeAction(t *testing.T) {
//	logrus.Infof("--- TestInvokeAction ---")
//	const actionValue = "1 2 3 action!"
//	var receivedAction int = 0
//	var rxMutex = sync.Mutex{}
//
//	// step 1 create the MQTT message bus client
//	client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
//	err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
//	assert.NoError(t, err)
//
//	// step 2 create a ConsumedThing and listen for actions on the mqtt bus
//	cThing := mqttbinding.Consume(testTD, client)
//	actionTopic := strings.ReplaceAll(mqttbinding.TopicInvokeAction, "{thingID}", testThingID) + "/#"
//	client.Subscribe(actionTopic, func(address string, message []byte) {
//		rxMutex.Lock()
//		defer rxMutex.Unlock()
//		receivedAction++
//		var rxData2 string
//		err := json.Unmarshal(message, &rxData2)
//		assert.NoError(t, err)
//		assert.Equal(t, actionValue, rxData2)
//	})
//
//	// step 3 publish the action
//	err = cThing.InvokeAction(testActionName, actionValue)
//	assert.NoError(t, err)
//	time.Sleep(time.Second)
//
//	// step 4 check result
//	rxMutex.Lock()
//	assert.Equal(t, 1, receivedAction)
//	defer rxMutex.Unlock()
//
//	// step 5 cleanup
//	cThing.Stop()
//	client.Unsubscribe(actionTopic)
//	client.Disconnect()
//}
//
//func TestInvokeActionBadName(t *testing.T) {
//	logrus.Infof("--- TestInvokeActionBadName ---")
//
//	// step 1 create the MQTT message bus client
//	client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
//	err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
//	assert.NoError(t, err)
//
//	// step 2 create a ConsumedThing and listen for actions on the mqtt bus
//	cThing := mqttbinding.Consume(testTD, client)
//
//	// step 3 publish the action with unknown name
//	err = cThing.InvokeAction("unknown-action", "")
//	assert.Error(t, err)
//
//	// step 4 cleanup
//	cThing.Stop()
//	client.Disconnect()
//}
//
//func TestWriteProperty(t *testing.T) {
//	const testNewPropValue1 = "new value 1"
//	const testNewPropValue2 = "new value 2"
//	logrus.Infof("--- TestWriteProperty ---")
//
//	// step 1 create the MQTT message bus client
//	client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
//	err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
//	assert.NoError(t, err)
//
//	// step 2 create a ConsumedThing
//	cThing := mqttbinding.Consume(testTD, client)
//
//	// step 3 submit the write request
//	err = cThing.WriteProperty(testProp1Name, testNewPropValue1)
//	assert.NoError(t, err)
//	time.Sleep(time.Second)
//
//	newProps := make(map[string]interface{})
//	newProps[testProp1Name] = testNewPropValue2
//	err = cThing.WriteMultipleProperties(newProps)
//	assert.NoError(t, err)
//	time.Sleep(time.Second)
//
//	// step 5 cleanup
//	cThing.Stop()
//	client.Disconnect()
//}
