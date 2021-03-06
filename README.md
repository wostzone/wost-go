# wost-go WoST Golang Library

This repository provides a library with definitions and methods to use WoST Hub services. It is intended for developing
IoT "Thing" devices and for developing consumers of Thing information.

## Summary

This Go library provides packages for building WoST services, IoT devices and clients, including:

- building TD (Thing Description) documents
- exposing Things for IoT devices
- consume Things for consumers
- authenticate using certificates, BASIC or JWT tokens
- discover services using DNS-SD
- managing certificates
- connecting to the MQTT message bus
- launch a test environment with a MQTT broker

## Packages

### config

Loading of Hub, service or device yaml configuration.

### certsclient

Management of keys
Loading and saving of TLS certificates

### config

Helper functions to load commandline and configuration files used to start a client and to configure logging.

Use:
> hubConfig, err := LoadAllConfig(os.args, "", clientID, &clientConfig)

To load the hub configuration and the custom client configuration from {clientID}.yaml

### consumedthing

ConsumedThing class for interacting with an exposed thing. ConsumedThing's are created using the ConsumedThingFactory
that provides the needed protocol bindings.
Consumed Things are defined in [WoT scripting API](https://w3c.github.io/wot-scripting-api/#the-consumedthing-interface)

### discovery

Client for discovery of services by their service name. This is used for example in the idprov provisioning client to
discover the provisioning server.

For example, to discover the URL of the idprov service:

```golang
   serviceName := "idprov"
address, port, paraMap, records, err := discovery.DiscoverServices(serviceName, 0)
```

### exposedthing

ExposedThing that represents an IoT device or service. ExposedThings are created using the ExposeThingFactory that
provides the needed protocol bindings.
Exposed Things are defined in
the [WoT scripting API](https://w3c.github.io/wot-scripting-api/#the-exposedthing-interface)

### hubnet

Helper functions for:

- Determine the outbound interface(s)
- Obtain bearer token for authentication

### logging

Standardized logging formatting using logrus. This includes the sourcefile name and line number.

### mqttclient

Client to connect to the Hub MQTT broker. The MQTT client is build around the paho mqtt client and adds reconnects, and
CA certificate verification with client certificate or username/password authentication.

The MqttHubClient includes publishing and subscribing to WoST messages such as Action, Config (properties), Events,
Property value updates and the full TD document. WoST Thing devices use these to publish their things and listen for
action requests.

For example, to connect to the message bus using a client certificate:

```golang
    client := mqttclient.NewMqttClient(testPluginID, certs.CaCert, 0)
err := client.ConnectWithClientCert(mqttCertAddress, certs.PluginCert)
```

### signing

> ### This section is subject to change
The signing package provides functions to JWS sign and JWE encrypt messages. This is used to verify the authenticity of
the sender of the message.

Signing and sender verification guarantees that the information has not been tampered with and originated from the
sender.

### thing

Definitions and functions to build a Thing Description document with properties, events and action affordances (
definitions).

Note: The generated TD is a best effort to conform to the WoT standard.

For example, to build a new TD of a temperature sensor with a temperature property:

```golang
    import "github.com/wostzone/hub/lib/client/pkg/thing"
import "github.com/wostzone/hub/lib/client/pkg/vocab"

...
thingID := CreateThingID("local", "publisher1", "device1", vocab.DeviceTypeSensor)
tdoc := thing.CreateTD(thingID, "Sensor", vocab.DeviceTypeSensor)
prop := tdoc.UpdateProperty("otemp", thing.PropertyAffordance{
Title:"Outdoor temperature",
Unit: vocab.UnitNameCelcius,
Type: vocab.WoTDataTypeNumber,
ReadOnly: true,
AtType: vocab.PropertyTypeTemperature})
tdoc.SetPropertyDataTypeInteger(prop, -100, 100)
```

Under consideration:

* Signing of messages. Most likely using JWS.
* Encryption of messages. Presumably using JWE. It can be useful for sending messages to the device that should not be
  accessible to others on the message bus.

## testenv

testenv simulates a server for testing of clients. This includes generating of certificates and setup and run a
mosquitto mqtt test server.

For example, to test a client with a mosquitto server using the given configuration and certificate folder for use by
mosquitto:

```golang
    certs = testenv.CreateCertBundle()
mosquittoCmd, err := testenv.StartMosquitto(configFolder, certFolder, &certs)
...run the tests...
testenv.StopMosquitto(mosquittoCmd)
```

See: pkg/mqttclient/MqttClient_test.go for examples

### tlsclient

TLSClient is a client for connecting to TLS servers such as the Hub's core ThingDirectory service. This client supports
both certificate and username/password authentication using JWT with refresh tokens.

For example, an IoT device can connect to a Hub service using its client certificate:

```golang
  caCert := LoadCertFromPem(pathToCACert)
clientCert := LoadCertFromPem(pathToClientCert)
client, err := tlsclient.NewTLSClient("host:port", caCert)
err = client.ConnectWithClientCert(clientCert)

// do stuff
client.Post(path, message)

client.Close()
```

### tlsserver

Server of HTTP/TLS connections that supports certificate and username/password authentication, and authorization.

Used to build Hub services that connect over HTTPS, such as the IDProv protocol server and the Thingdir directory
server.

## vocab

Ontology with vocabulary used to describe Things. This is based on terminology from the WoT working group and other
source.

When no authorative source is known, the terminology is defined as part of the WoST IoT vocabulary. This includes
device-type names, Thing property types, property names, unit names and TD defined terms for describing a Thing
Description document.

### watcher

Simple file watcher that handles renaming of files.
