// Package discovery to publish Hub services for discovery
package discovery

import (
	"fmt"
	"net"
	"os"

	"github.com/wostzone/wost-go/pkg/hubnet"

	"github.com/grandcat/zeroconf"
	"github.com/sirupsen/logrus"
)

// DiscoServe publishes a WoST service for discovery.
// See also 'DiscoClient' for discovery of this published service.
//
// WoST services use this to announce the service instance and how they can be reached on the local domain.
//   Instance = instance name of the service. Used to differentiate between services with the same name (type)
//   DiscoveryServiceName = name of the provided service, for example, ipp, idprov, wotdir
//
// This is a wrapper around one or more discovery methods. Internally this uses DNS-SD but can be
// expanded with additional protocols in the future.
//  DNS-SD will publish this as _<instance>._<serviceName>._tcp
//
//  instanceID is the unique ID of the service instance, usually the plugin-ID
//  serviceName is the discover name. For example "idprov"
//  address service listening IP address
//  port service listing port
//  params is a map of key-value pairs to include in discovery, for example {path:/idprov/}
// Returns the discovery service instance. Use Shutdown() when done.
func DiscoServe(instanceID string, serviceName string,
	address string, port uint, params map[string]string) (*zeroconf.Server, error) {
	var ips []string

	logrus.Infof("DiscoServe serviceID=%s, name=%s, address: %s:%d, params=%s",
		instanceID, serviceName, address, port, params)
	if serviceName == "" {
		err := fmt.Errorf("DiscoServe: empty serviceName")
		return nil, err
	}

	// only the local domain is supported
	domain := "local."
	hostname, _ := os.Hostname()

	// if the given address isn't a valid IP address. try to resolve it instead
	ips = []string{address}
	if net.ParseIP(address) == nil {
		// was a hostname provided instead IP?
		hostname = address
		actualIP, err := net.LookupIP(address)
		if err != nil {
			// can't continue without a valid address
			logrus.Errorf("DiscoServe: Provided address '%s' is not an IP and cannot be resolved: %s", address, err)
			return nil, err
		}
		ips = []string{actualIP[0].String()}
	}

	ifaces, err := hubnet.GetInterfaces(ips[0])
	if err != nil || len(ifaces) == 0 {
		logrus.Warningf("DiscoServe: Address %s does not appear on any interface. Continuing anyways", ips[0])
	}
	// add a text record with key=value pairs
	textRecord := []string{}
	for k, v := range params {
		textRecord = append(textRecord, fmt.Sprintf("%s=%s", k, v))
	}
	// I don't like this 'hiding' of the service type, but it is too DNS-SD specific
	serviceType := fmt.Sprintf("_%s._tcp", serviceName)
	server, err := zeroconf.RegisterProxy(
		instanceID, serviceType, domain, int(port), hostname, ips, textRecord, ifaces)
	if err != nil {
		logrus.Errorf("DiscoServe: Failed to start the zeroconf server: %s", err)
	}
	return server, err
}
