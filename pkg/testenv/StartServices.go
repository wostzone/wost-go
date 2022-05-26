// Package testenv with simulated services for authentication and directory
package testenv

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/wostzone/wost-go/pkg/tlsclient"
	"net/http"
)

const testAddress = "localhost"
const testPort = 9882

// StartServices starts a TLS server and listens for auth and dir requests
func StartServices(certs *TestCerts) *http.Server {
	router := mux.NewRouter()
	// service has CA certificate for client cert authentication
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(certs.CaCert)

	// service serves the TLS server certificate
	tlsConf := &tls.Config{
		Certificates:       []tls.Certificate{*certs.ServerCert},
		ClientAuth:         tls.VerifyClientCertIfGiven,
		ClientCAs:          caCertPool,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}

	httpServer := &http.Server{
		Addr: fmt.Sprintf("%s:%d", testAddress, testPort),
		// ReadTimeout:  5 * time.Minute, // 5 min to allow for delays when 'curl' on OSx prompts for username/password
		// WriteTimeout: 10 * time.Second,
		Handler:   router,
		TLSConfig: tlsConf,
	}
	router.HandleFunc(tlsclient.DefaultJWTLoginPath, func(resp http.ResponseWriter, req *http.Request) {
	})
	router.HandleFunc(tlsclient.DefaultJWTRefreshPath, func(resp http.ResponseWriter, req *http.Request) {
	})

	go func() {
		// serverTLSConf contains certificate and key
		httpServer.ListenAndServeTLS("", "")
	}()
	return httpServer
}
