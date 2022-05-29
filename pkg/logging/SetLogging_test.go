package logging_test

import (
	"github.com/wostzone/wost-go/pkg/logging"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLogging(t *testing.T) {
	//wd, _ := os.Getwd()
	//logFile := path.Join(wd, "../../test/logs/TestLogging.log")
	logFile := ""

	os.Remove(logFile)
	logging.SetLogging("info", logFile)
	logrus.Info("Hello info")
	logging.SetLogging("debug", logFile)
	logrus.Debug("Hello debug")
	logging.SetLogging("warn", logFile)
	logrus.Warn("Hello warn")
	logging.SetLogging("error", logFile)
	logrus.Error("Hello error")
	//assert.FileExists(t, logFile)
	//os.Remove(logFile)
}
