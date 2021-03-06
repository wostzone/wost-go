package discovery_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wostzone/wost-go/pkg/discovery"
	"github.com/wostzone/wost-go/pkg/hubnet"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testServiceID = "discovery-test"
const testServiceName = "test-service"
const testServicePath = "/discovery/path"
const testServicePort = uint(9999)

// Test the discovery client and server
func TestDiscover(t *testing.T) {
	params := map[string]string{"path": testServicePath}
	testServiceAddress := hubnet.GetOutboundIP("").String()

	discoServer, err := discovery.DiscoServe(
		testServiceID, testServiceName, testServiceAddress, testServicePort, params)

	assert.NoError(t, err)
	assert.NotNil(t, discoServer)

	// Test if it is discovered
	address, port, discoParams, records, err := discovery.DiscoClient(testServiceName, 1)
	require.NoError(t, err)
	rec0 := records[0]
	assert.Equal(t, testServiceID, rec0.Instance)
	assert.Equal(t, testServiceAddress, address)
	assert.Equal(t, testServicePort, port)
	assert.Equal(t, testServicePath, discoParams["path"])

	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}

func TestDiscoViaDomainName(t *testing.T) {
	testServiceAddress := "localhost"

	discoServer, err := discovery.DiscoServe(
		testServiceID, testServiceName, testServiceAddress, testServicePort, nil)

	assert.NoError(t, err)
	assert.NotNil(t, discoServer)

	// Test if it is discovered
	discoAddress, discoPort, _, records, err := discovery.DiscoClient(testServiceName, 1)
	rec0 := records[0]
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", discoAddress)
	assert.True(t, strings.HasPrefix(rec0.HostName, testServiceAddress))
	assert.Equal(t, testServicePort, discoPort)

	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}

func TestDiscoverBadPort(t *testing.T) {
	serviceID := "idprov-test"
	badPort := uint(0)
	address := hubnet.GetOutboundIP("").String()
	_, err := discovery.DiscoServe(
		serviceID, testServiceName, address, badPort, nil)

	assert.Error(t, err)
}

func TestNoInstanceID(t *testing.T) {
	serviceID := "serviceID"
	address := hubnet.GetOutboundIP("").String()

	_, err := discovery.DiscoServe(
		"", testServiceName, address, testServicePort, nil)
	assert.Error(t, err) // missing instance name

	_, err = discovery.DiscoServe(
		serviceID, "", address, testServicePort, nil)
	assert.Error(t, err) // missing service name
}

func TestDiscoverNotFound(t *testing.T) {
	instanceID := "idprov-test-id"
	serviceName := "idprov-test"
	address := hubnet.GetOutboundIP("").String()

	discoServer, err := discovery.DiscoServe(
		instanceID, serviceName, address, testServicePort, nil)

	assert.NoError(t, err)

	// Test if it is discovered
	discoAddress, discoPort, _, records, err := discovery.DiscoClient("wrongname", 1)
	_ = discoAddress
	_ = discoPort
	_ = records
	assert.Error(t, err)

	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
	assert.Error(t, err)
}

func TestBadAddress(t *testing.T) {
	instanceID := "idprov-test-id"

	discoServer, err := discovery.DiscoServe(
		instanceID, testServiceName, "notanipaddress", testServicePort, nil)

	assert.Error(t, err)
	assert.Nil(t, discoServer)
}

func TestExternalAddress(t *testing.T) {
	instanceID := "idprov-test-id"

	discoServer, err := discovery.DiscoServe(
		instanceID, testServiceName, "1.2.3.4", testServicePort, nil)

	// expect a warning
	assert.NoError(t, err)
	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}

func TestDNSSDScan(t *testing.T) {

	records, err := discovery.DnsSDScan("", 2)
	fmt.Printf("Found %d records in scan", len(records))

	assert.NoError(t, err)
	assert.Greater(t, len(records), 0, "No DNS records found")
}
