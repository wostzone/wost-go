package config

import (
	"flag"
	"os"
	"path"

	"github.com/sirupsen/logrus"
)

// LoadAllConfig is a helper to load all configuration from commandline, hubconfig and client config
// This:
//  1. Determine application defaults
//  2. parse commandline arguments for options -c hub.yaml -a appFolder or -h
//  3. Load the hub global configuration file hub.yaml, if found
//  4. Load the client configuration file {clientID}.yaml, if found
//
//  args is the os.argv list. Use nil to ignore commandline args
//  homeFolder is the installation folder, "" for default parent folder of app binary
//  clientID is the server, plugin or device instance ID. Used when connecting to servers
//  clientConfig is an instance of the client's configuration object
// This returns the hub global configuration with an error if something went wrong
func LoadAllConfig(args []string, homeFolder string, clientID string, clientConfig interface{}) (*HubConfig, error) {
	hubConfigFile := DefaultHubConfigName

	// Determine the default application installation folder
	if homeFolder == "" {
		appBin, _ := os.Executable()
		binFolder := path.Dir(appBin)
		homeFolder = path.Dir(binFolder)
	}

	// Parse commandline arguments for options -c configFile and -a homeFolder
	if args != nil {
		//var cmdHubConfigFile string
		//var cmdAppFolder string
		flag.StringVar(&hubConfigFile, "c", hubConfigFile, "Change the global hub configuration file")
		flag.StringVar(&homeFolder, "a", homeFolder, "Change the application home folder with config and cert subfolders")
		flag.Parse()
	}
	hubConfig := CreateHubConfig(homeFolder)
	err := hubConfig.Load(hubConfigFile, clientID)
	if err != nil {
		logrus.Errorf("LoadConfig: Hub config file '%s' failed to load: %s", hubConfigFile, err)
		return hubConfig, err
	}
	// last the client settings (optional)
	if clientConfig != nil {
		clientConfigFile := path.Join(hubConfig.ConfigFolder, clientID+".yaml")
		substituteMap := make(map[string]string)
		substituteMap["{clientID}"] = clientID
		substituteMap["{homeFolder}"] = hubConfig.HomeFolder
		substituteMap["{configFolder}"] = hubConfig.ConfigFolder
		substituteMap["{logsFolder}"] = hubConfig.LogFolder
		substituteMap["{certsFolder}"] = hubConfig.CertsFolder

		if _, err = os.Stat(clientConfigFile); os.IsNotExist(err) {
			logrus.Infof("FYI The optional client configuration file %s is not present", clientConfigFile)
			err = nil
		} else {
			err = LoadYamlConfig(clientConfigFile, clientConfig, substituteMap)
		}
	}
	return hubConfig, err
}
