// Package testenv for creating mosquitto testing environment
// This requires that the mosquitto broker is installed.
package testenv

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MQTT ports for test environment in the 9xxx range
const (
	MqttPortUnpw = 9883
	MqttPortCert = 9884
	MqttPortWS   = 9885
)
const (
	caCertFile     = "caCert.pem"
	caKeyFile      = "caKey.pem"
	serverCertFile = "serverCert.pem"
	serverKeyFile  = "serverKey.pem"
	pluginCertFile = "pluginCert.pem"
	pluginKeyFile  = "pluginKey.pem"
)

const mosquittoConfigFile = "mosquitto-testsetup.conf"

const mqConfigTemplate = `
# Generated by 'testenv.StartMosquitto.go'. Do not edit.

log_type error
log_type warning
log_type information
log_dest stdout
log_dest stderr
persistence false

#--- plugins and devices use certificates with MQTT
# MQTT over TLS/SSL
listener {{.mqttPortCert}}
require_certificate true
tls_version tlsv1.2
cafile {{.certFolder}}/{{.caCertFile}}
certfile {{.certFolder}}/{{.serverCertFile}}
keyfile {{.certFolder}}/{{.serverKeyFile}}

#--- consumers using username/pw with MQTT protocol over TLS/SSL
listener {{.mqttPortUnpw}}
require_certificate false
tls_version tlsv1.2
cafile {{.certFolder}}/{{.caCertFile}}
certfile {{.certFolder}}/{{.serverCertFile}}
keyfile {{.certFolder}}/{{.serverKeyFile}}
# No password needed for users while testing
allow_anonymous true

#--- consumers use username/pw with WebSockets over TLS/SSL
listener {{.mqttPortWS}}
protocol websockets
require_certificate false
tls_version tlsv1.2
cafile {{.certFolder}}/{{.caCertFile}}
certfile {{.certFolder}}/{{.serverCertFile}}
keyfile {{.certFolder}}/{{.serverKeyFile}}
# No password needed for users while testing
allow_anonymous true
`

// CreateMosquittoConf creates a mosquitto.conf file for testing
func createMosquittoConf(configFolder string, certFolder string) string {
	var output bytes.Buffer
	params := map[string]string{
		"configFolder":   configFolder,
		"certFolder":     certFolder,
		"caCertFile":     caCertFile,
		"serverCertFile": serverCertFile,
		"serverKeyFile":  serverKeyFile,
		"mqttPortCert":   fmt.Sprint(MqttPortCert),
		"mqttPortWS":     fmt.Sprint(MqttPortWS),
		"mqttPortUnpw":   fmt.Sprint(MqttPortUnpw),
	}
	confTpl, _ := template.New("").Parse(mqConfigTemplate)
	confTpl.Execute(&output, params)
	return output.String()
	// return ""
}

// StartMosquitto create a test environment with a mosquitto broker on localhost for the given home folder
// This:
//  1. Set logging to info
//  2. create the cert/config folder if it doesn't exist
//  3. Saves the CA, server and client certificates in the cert/config folder
//  4. Generates a mosquitto configuration in the cert/config folder
//  5. Launches a mosquitto broker for testing.
//
// mqCmd.Process.Kill() to end the mosquitto broker
//
//  testCerts are the certificates to use.
//  configFolder to store certificates and configuration. Will be created if it doesn't exist.
// Returns the mosquitto process, the temp folder for cleanup and error code in case of failure
func StartMosquitto(testCerts *TestCerts, configFolder string) (mqCmd *exec.Cmd, err error) {
	mutex := sync.Mutex{}

	logrus.Infof("--- Starting mosquitto broker ---")
	if configFolder == "" {
		configFolder, _ = ioutil.TempDir("", "wost-go-testenv")
	}

	// mqCmd = Launch(mosqConfigPath)
	SaveCerts(testCerts, configFolder)
	// mosquitto must be in the path to execute
	mosqConf := createMosquittoConf(configFolder, configFolder)
	mosqConfigPath := path.Join(configFolder, mosquittoConfigFile)
	err = ioutil.WriteFile(mosqConfigPath, []byte(mosqConf), 0644)
	if err != nil {
		logrus.Fatalf("Setup: Unable to write mosquitto config file: %s", err)
	}
	mqCmd = exec.Command("mosquitto", "-c", mosqConfigPath)
	// Capture stderr in case of startup failure
	mqCmd.Stderr = os.Stderr
	mqCmd.Stdout = os.Stdout
	mqCmd.Start()
	go func() {
		err2 := mqCmd.Wait()
		mutex.Lock()
		err = err2
		mutex.Unlock()
		logrus.Infof("--- Mosquitto has ended ---")
	}()
	// Give mosquitto some time to start
	time.Sleep(100 * time.Millisecond)
	mutex.Lock()
	defer mutex.Unlock()
	if err != nil {
		logrus.Fatalf("Failed starting mosquitto: %s", err)
	}
	return mqCmd, err
}

// StopMosquitto stops the mosquitto broker and cleans up the test environment
//  cmd is the command returned by StartMosquitto
//  tempFolder is the folder returned by StartMosquitto. This will be deleted. Use "" to keep it
func StopMosquitto(cmd *exec.Cmd, tempFolder string) {
	logrus.Infof("--- Stopping mosquitto broker ---")
	cmd.Process.Signal(os.Interrupt)
	time.Sleep(100 * time.Millisecond)

	if tempFolder != "" {
		os.RemoveAll(tempFolder)
	}
}
