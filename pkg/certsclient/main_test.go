package certsclient_test

import (
	"github.com/wostzone/wost-go/pkg/logging"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

var testCertFolder string
var testPrivKeyPemFile string
var testPubKeyPemFile string

// TestMain create a test folder for certificates and private key
func TestMain(m *testing.M) {
	testCertFolder, _ = ioutil.TempDir("", "wost-go-")

	testPrivKeyPemFile = path.Join(testCertFolder, "privKey.pem")
	testPubKeyPemFile = path.Join(testCertFolder, "pubKey.pem")
	logging.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
		println("Find test files in:", testCertFolder)
	} else {
		// comment out the next line to be able to inspect results
		os.RemoveAll(testCertFolder)
	}

	os.Exit(result)
}
