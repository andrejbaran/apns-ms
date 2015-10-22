// Example of a command line executable for running APNS Provider as "microservice" but despite it being an example it can be used out of the box as is.
//
// It is meant as a "microservice" thus it exposes HTTP API for communication with 3rd party programs.
//
// Usage
//
// List all available options:
//  apns --help
//
// Available options:
//   --apns-gate-port=2195: Apple's APNS port number
//   --apns-gate-production="gateway.push.apple.com": FQDN of Apple's APNS production gateway.
//   --apns-gate-sandbox="gateway.sandbox.push.apple.com": FQDN of Apple's APNS sandbox gateway.
//   --bind-address=0.0.0.0: IP address the HTTP server should bind to.
//   --bind-port=9090: Port on which HTTP server is listening.
//   --cert="": Absolute path to certificate file. Certificate is expected be in PEM format.
//   --cert-key="": Absolute path to certificate private key file. Certificate key is expected be in PEM format.
//   --env="sandbox": Environment of Apple's APNS and Feedback service gateways. For production use specify "production", for testing specify "sandbox".
//   --expired-devices-endpoint="/expired-devices": URI of Expired device tokens endpoint.
//   --feedback-gate-port=2196: Apple's Feedback service port number
//   --feedback-gate-production="feedback.push.apple.com": FQDN of Apple's Feedback service production gateway.
//   --feedback-gate-sandbox="feedback.sandbox.push.apple.com": FQDN of Apple's Feedback service sandbox gateway.
//   --max-notifications=100000: Number of notification that can be queued for processing at once. Once the queue is full all requests to raw push notification endpoint will result in 503 Service Unavailable response.
//   --notification-endpoint="/notification": URI of Raw push notification endpoint.
//   --workers=4: Number of workers that concurently process push notifications. Defaults to 2 * Number of CPU cores.
//
//
package main

import (
	"apns-microservice/apns"
	"apns-microservice/server"
	"fmt"
	log "github.com/coreos/pkg/capnslog"
	"github.com/spf13/pflag"
	"net/http"
	"os"
)

var apnsLogger, serverLogger *log.PackageLogger

func init() {
	log.SetFormatter(log.NewPrettyFormatter(os.Stdout, true))
	apnsLogger = log.NewPackageLogger("apns-microservice", "apns")
	serverLogger = log.NewPackageLogger("apns-microservice", "http")

	log.SetGlobalLogLevel(log.INFO)

	apns.SetLogger(apnsLogger)
	server.SetLogger(serverLogger)
}

func main() {
	apns.SetupCommandLineFlags(pflag.CommandLine)
	server.SetupCommandLineFlags(pflag.CommandLine)
	pflag.Parse()

	config := apns.NewClientConfig()
	client, err := apns.NewClient(config)
	if err != nil {
		return
	}

	http.HandleFunc(server.RawNotificationEndpoint, server.NewRawNotificationHTTPHandlerFunc(client))
	http.HandleFunc(server.ExpiredDeviceTokensEndpoint, server.NewExpiredDevicesHTTPHandlerFunc(client))

	serverLogger.Infof("Starting server %s:%d", server.Address.String(), server.Port)

	serverErr := http.ListenAndServe(fmt.Sprintf("%s:%d", server.Address.String(), server.Port), nil)
	if serverErr != nil {
		serverLogger.Fatalf("Server failed to start: %s", serverErr)
	}
}
