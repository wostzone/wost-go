// Package config with the global hub configuration struct and methods
package config

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/wostzone/wost-go/pkg/certsclient"
	"github.com/wostzone/wost-go/pkg/hubnet"
)

// DefaultHubConfigName with the configuration file name of the hub
const DefaultHubConfigName = "hub.yaml"

// DefaultBinFolder is the location of application binaries wrt installation folder
const DefaultBinFolder = "./bin"

// DefaultCertsFolder with the location of certificates
const DefaultCertsFolder = "./certs"

// DefaultConfigFolder is the location of config files wrt installation folder
const DefaultConfigFolder = "./config"

// DefaultLogFolder is the location of log files wrt installation folder
const DefaultLogFolder = "./log"

// auth
// const (
// 	DefaultAclFile  = "hub.acl"
// 	DefaultUnpwFile = "hub.passwd"
// )

// Default ports for connecting to the MQTT server
const (
	DefaultMqttPortUnpw = 8883
	DefaultMqttPortCert = 8884
	DefaultMqttPortWS   = 8885
)
const DefaultMqttTimeout = 3

// Default certificate and private key file names
const (
	DefaultCaCertFile     = "caCert.pem"
	DefaultCaKeyFile      = "caKey.pem"
	DefaultPluginCertFile = "pluginCert.pem"
	DefaultPluginKeyFile  = "pluginKey.pem"
	DefaultServerCertFile = "serverCert.pem"
	DefaultServerKeyFile  = "serverKey.pem"
	DefaultAdminCertFile  = "adminCert.pem"
	DefaultAdminKeyFile   = "adminKey.pem"
)

// DefaultThingZone is the zone the hub's things are published in.
const DefaultThingZone = "local"

// HubConfig contains the global configuration for using the Hub by its clients.
//
// Intended for use by:
//  1. Hub plugins that needs to know the location of files, certificates and service address and ports
//  2. Remote devices or services that uses a local copy of the hub config for manual configuration of
//     certificates MQTT server address and ports.
//
type HubConfig struct {

	// Server address of auth, idprov, mqtt, directory services. The default "" is the outbound IP address
	// If DNS is used override this with the server domain name.
	// Clients that use the hubconfig must override this with the discovered server address as provided by idprov
	Address string `yaml:"address,omitempty"`
	// MQTT TLS port for certificate based authentication. Default is DefaultMqttPortCert
	MqttPortCert int `yaml:"mqttPortCert,omitempty"`
	// MQTT TLS port for login/password authentication. Default is DefaultMqttPortUnpw
	MqttPortUnpw int `yaml:"mqttPortUnpw,omitempty"`
	// Websocket TLS port for login/password authentication. Default is DefaultMqttPortWS
	MqttPortWS int `yaml:"mqttPortWS,omitempty"`
	// plugin mqtt connection timeout in seconds. 0 for indefinite. Default is DefaultMqttTimeout (3 sec)
	MqttTimeout int `yaml:"mqttTimeout,omitempty"`

	// auth
	// AclStorePath  string `yaml:"aclStore"`  // path to the ACL store
	// UnpwStorePath string `yaml:"unpwStore"` // path to the uername/password store

	// Zone that published Things belong to. Default is 'local'
	// Zones are useful for separating devices from Hubs on large networks. Normally 'local' is sufficient.
	//
	// When Things are bridged, the bridge can be configured to replace the zone by that of the bridge.
	// This is intended for access control to Things from a different zone.
	Zone string `yaml:"zone"`

	// Files and Folders
	Loglevel    string `yaml:"logLevel"`    // debug, info, warning, error. Default is warning
	LogFolder   string `yaml:"logFolder"`   // location of Wost log files
	LogFile     string `yaml:"logFile"`     // log filename is pluginID.log
	HomeFolder  string `yaml:"homeFolder"`  // Folder containing the application installation
	BinFolder   string `yaml:"binFolder"`   // Folder containing plugin binaries, default is {homeFolder}/bin
	CertsFolder string `yaml:"certsFolder"` // Folder containing certificates, default is {homeFolder}/certsclient
	// ConfigFolder the location of additional configuration files. Default is {homeFolder}/config
	ConfigFolder string `yaml:"configFolder"`

	// Keep server certificate on startup. Default is false
	// enable to keep using access tokens between restarts
	KeepServerCertOnStartup bool `yaml:"keepServerCertOnStartup"`

	// path to CA certificate in PEM format. Default is homeFolder/certs/caCert.pem
	CaCertFile string `yaml:"caCertFile"`
	// path to client x509 certificate in PEM format. Default is homeFolder/certs/{clientID}Cert.pem
	ClientCertFile string `yaml:"clientCertFile"`
	// path to client private key in PEM format. Default is homeFolder/certs/{clientID}Key.pem
	ClientKeyFile string `yaml:"clientKeyFile"`
	// path to plugin client x509 certificate in PEM format. Default is homeFolder/certs/PluginCert.pem
	PluginCertFile string `yaml:"pluginCertFile"`
	// path to plugin client private key in PEM format. Default is homeFolder/certs/PluginKey.pem
	PluginKeyFile string `yaml:"pluginKeyFile"`

	// CaCert contains the loaded CA certificate needed for establishing trusted connections to the
	// MQTT message bus and other services. Loading takes place in LoadHubConfig()
	CaCert *x509.Certificate

	// ClientCert contains the loaded TLS client certificate and key if available.
	// Loading takes place in LoadHubConfig()
	// * For plugins this is the plugin certificate and private key
	// * For servers this is the server certificate and private key
	// * For devices this is the provisioned device certificate and private key
	ClientCert *tls.Certificate

	// PluginCert contains the TLS client certificate for use by plugins
	// Intended for use by plugin clients. This is nil of the plugin certificate is not available or accessible
	// Loading takes place in LoadHubConfig()
	PluginCert *tls.Certificate
}

// AsMap returns a key-value map of the HubConfig
// This simply converts the yaml to a map
func (hubConfig *HubConfig) AsMap() map[string]string {
	kvMap := make(map[string]string)
	encoded, _ := yaml.Marshal(hubConfig)
	_ = yaml.Unmarshal(encoded, &kvMap)
	return kvMap
}

// Load loads and validates the configuration from file.
//
// If an error is returned then the default configuration is returned.
//
// The following variables can be used in this file:
//    {clientID}  is the device or plugin instance ID. Used for logfile and client cert
//    {homeFolder} is the default application folder (parent of application binary)
//    {certsFolder} is the default certificate folder
//    {configFolder} is the default configuration folder
//    {logFolder} is the default logging folder
//
//  configFile is optional. The default is hub.yaml in the default config folder.
//  clientID is the device or plugin instance ID. Used for logfile and client cert name.
//
// Returns the hub configuration and error code
//
func (hubConfig *HubConfig) Load(configFile string, clientID string) error {

	substituteMap := make(map[string]string)
	substituteMap["{clientID}"] = clientID
	substituteMap["{homeFolder}"] = hubConfig.HomeFolder
	substituteMap["{configFolder}"] = hubConfig.ConfigFolder
	substituteMap["{logFolder}"] = hubConfig.LogFolder
	substituteMap["{certsFolder}"] = hubConfig.CertsFolder

	// make sure the config file path is absolute
	if configFile == "" {
		configFile = path.Join(hubConfig.ConfigFolder, DefaultHubConfigName)
	} else if !path.IsAbs(configFile) {
		configFile = path.Join(hubConfig.ConfigFolder, configFile)
	}

	logrus.Infof("Using %s as hub config file", configFile)
	err := LoadYamlConfig(configFile, hubConfig, substituteMap)
	if err != nil {
		return err
	}

	// make sure files and folders have an absolute path
	if !path.IsAbs(hubConfig.CertsFolder) {
		hubConfig.CertsFolder = path.Join(hubConfig.HomeFolder, hubConfig.CertsFolder)
	}

	if !path.IsAbs(hubConfig.LogFolder) {
		hubConfig.LogFolder = path.Join(hubConfig.HomeFolder, hubConfig.LogFolder)
	}

	if hubConfig.LogFile == "" {
		hubConfig.LogFile = path.Join(hubConfig.LogFolder, clientID+".log")
	} else if !path.IsAbs(hubConfig.LogFile) {
		hubConfig.LogFile = path.Join(hubConfig.LogFolder, hubConfig.LogFile)
	}

	if !path.IsAbs(hubConfig.ConfigFolder) {
		hubConfig.ConfigFolder = path.Join(hubConfig.HomeFolder, hubConfig.ConfigFolder)
	}

	// CA certificate for use by everyone
	if hubConfig.CaCertFile == "" {
		hubConfig.CaCertFile = path.Join(hubConfig.CertsFolder, DefaultCaCertFile)
	} else if !path.IsAbs(hubConfig.CaCertFile) {
		hubConfig.CaCertFile = path.Join(hubConfig.CertsFolder, hubConfig.CaCertFile)
	}

	// Plugin client certificate for use by plugin clients
	if hubConfig.PluginCertFile == "" {
		hubConfig.PluginCertFile = path.Join(hubConfig.CertsFolder, DefaultPluginCertFile)
	} else if !path.IsAbs(hubConfig.PluginCertFile) {
		hubConfig.PluginCertFile = path.Join(hubConfig.CertsFolder, hubConfig.PluginCertFile)
	}
	if hubConfig.PluginKeyFile == "" {
		hubConfig.PluginKeyFile = path.Join(hubConfig.CertsFolder, DefaultPluginKeyFile)
	} else if !path.IsAbs(hubConfig.PluginKeyFile) {
		hubConfig.PluginKeyFile = path.Join(hubConfig.CertsFolder, hubConfig.PluginKeyFile)
	}

	// Client certificate for use by clients with their own certificate, eg iot devices
	if hubConfig.ClientCertFile == "" {
		hubConfig.ClientCertFile = path.Join(hubConfig.CertsFolder, clientID+"Cert.pem")
	} else if !path.IsAbs(hubConfig.ClientCertFile) {
		hubConfig.ClientCertFile = path.Join(hubConfig.CertsFolder, hubConfig.ClientCertFile)
	}

	if hubConfig.ClientKeyFile == "" {
		hubConfig.ClientKeyFile = path.Join(hubConfig.CertsFolder, clientID+"Key.pem")
	} else if !path.IsAbs(hubConfig.ClientCertFile) {
		hubConfig.ClientKeyFile = path.Join(hubConfig.CertsFolder, hubConfig.ClientKeyFile)
	}

	// Certificate are optional as they might not yet exist
	hubConfig.CaCert, err = certsclient.LoadX509CertFromPEM(hubConfig.CaCertFile)
	if err != nil {
		logrus.Warningf("Unable to load the CA Certificate: %s. This is unexpected but continuing for now.", err)
	}
	// optional user client certificate, if available
	hubConfig.ClientCert, err = certsclient.LoadTLSCertFromPEM(hubConfig.ClientCertFile, hubConfig.ClientKeyFile)
	if err != nil && clientID != "" {
		// only warn if a client ID was given
		logrus.Warningf("Unable to load the Client Certificate: %s. This might not be needed so continuing for now.", err)
	}
	// optional plugin (client) certificate, if available
	hubConfig.PluginCert, err = certsclient.LoadTLSCertFromPEM(hubConfig.PluginCertFile, hubConfig.PluginKeyFile)
	if err != nil {
		logrus.Warningf("Unable to load the Plugin Certificate: %s. This is only needed for plugins so continuing for now.", err)
	}

	// validate the result
	err = hubConfig.Validate()
	return err
}

// Validate checks if the config, log, and certs folders in the hub configuration exist.
// Returns an error if the config is invalid
func (hubConfig *HubConfig) Validate() error {
	if _, err := os.Stat(hubConfig.HomeFolder); os.IsNotExist(err) {
		logrus.Errorf("Home folder '%s' not found\n", hubConfig.HomeFolder)
		return err
	}
	if _, err := os.Stat(hubConfig.ConfigFolder); os.IsNotExist(err) {
		logrus.Errorf("Configuration folder '%s' not found\n", hubConfig.ConfigFolder)
		return err
	}

	if _, err := os.Stat(hubConfig.LogFolder); os.IsNotExist(err) {
		logrus.Errorf("Logging folder '%s' not found\n", hubConfig.LogFolder)
		return err
	}

	if _, err := os.Stat(hubConfig.CertsFolder); os.IsNotExist(err) {
		logrus.Errorf("TLS certificate folder '%s' not found\n", hubConfig.CertsFolder)
		return err
	}

	return nil
}

// CreateHubConfig creates the HubConfig with default values
//
//  homeFolder is the hub installation folder and home to plugins, logs and configuration folders.
// Use "" for default: parent of application binary
// When relative path is given, it is relative to the current working directory (commandline use)
//
// See also LoadHubConfig to load the actual configuration including certificates.
func CreateHubConfig(homeFolder string) *HubConfig {
	appBin, _ := os.Executable()
	binFolder := path.Dir(appBin)
	if homeFolder == "" {
		homeFolder = path.Dir(binFolder)
	} else if !path.IsAbs(homeFolder) {
		// turn relative home folder in absolute path
		cwd, _ := os.Getwd()
		homeFolder = path.Join(cwd, homeFolder)
	}
	logrus.Infof("AppBin is: %s; Home is: %s", appBin, homeFolder)
	config := &HubConfig{
		HomeFolder:   homeFolder,
		BinFolder:    path.Join(homeFolder, DefaultBinFolder),
		CertsFolder:  path.Join(homeFolder, DefaultCertsFolder),
		ConfigFolder: path.Join(homeFolder, DefaultConfigFolder),
		LogFolder:    path.Join(homeFolder, DefaultLogFolder),
		Loglevel:     "warning",

		Address:      hubnet.GetOutboundIP("").String(),
		MqttPortCert: DefaultMqttPortCert,
		MqttPortUnpw: DefaultMqttPortUnpw,
		MqttPortWS:   DefaultMqttPortWS,
		// Plugins:      make([]string, 0),
		Zone: "local",
	}
	// config.Messenger.CertsFolder = path.Join(homeFolder, "certsclient")
	// config.AclStorePath = path.Join(config.ConfigFolder, DefaultAclFile)
	// config.UnpwStorePath = path.Join(config.ConfigFolder, DefaultUnpwFile)
	return config
}
