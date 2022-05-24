package exposedthing_test

import (
	"github.com/wostzone/wost-go/pkg/logging"
	"github.com/wostzone/wost-go/pkg/thing"
	"github.com/wostzone/wost-go/pkg/vocab"
	"os"
	"testing"
)

const testActionName = "action1"
const testEventName = "event1"
const testDeviceID = "device1"
const testDeviceType = vocab.DeviceTypeButton
const testProp1Name = "prop1"
const testProp1Value = "value1"

// The factory for consumed thing
//var factory *ConsumedThingFactory

// Create a test TD for use in consumed things
func createTestTD() *thing.ThingTD {
	title := "test Thing"
	thingID := thing.CreateThingID("", testDeviceID, testDeviceType)
	tdDoc := thing.CreateTD(thingID, title, testDeviceType)
	//
	prop1 := &thing.PropertyAffordance{
		DataSchema: thing.DataSchema{
			Type:  vocab.WoTDataTypeBool,
			Title: "Property 1",
		},
	}
	prop2 := &thing.PropertyAffordance{
		DataSchema: thing.DataSchema{
			Type:  vocab.WoTDataTypeBool,
			Title: "Event property",
		},
	}
	tdDoc.UpdateProperty(testProp1Name, prop1)
	tdDoc.UpdateProperty(testEventName, prop2)

	// add event to TD
	tdDoc.UpdateEvent(testEventName, &thing.EventAffordance{
		Data: thing.DataSchema{},
	})

	// add action to TD
	tdDoc.UpdateAction(testActionName, &thing.ActionAffordance{
		//Input: StringSchema{},
		Safe:       true,
		Idempotent: true,
	})
	return tdDoc
}

// TestMain - setup a test environment for testing consumed things
func TestMain(m *testing.M) {
	//factory = CreateConsumedThingFactory()

	//cwd, _ := os.Getwd()
	//homeFolder = path.Join(cwd, "../../test")
	//configFolder = path.Join(homeFolder, "config")
	//certFolder := path.Join(homeFolder, "certs")
	//_ = os.Chdir(homeFolder)
	//
	logging.SetLogging("info", "")
	//certs = testenv.CreateCertBundle()
	//mosquittoCmd, err := testenv.StartMosquitto(configFolder, certFolder, &certs)
	//if err != nil {
	//	logrus.Fatalf("Unable to start mosquitto: %s", err)
	//}
	//
	result := m.Run()
	//testenv.StopMosquitto(mosquittoCmd)
	os.Exit(result)
}
