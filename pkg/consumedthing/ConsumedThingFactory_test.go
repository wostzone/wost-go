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
