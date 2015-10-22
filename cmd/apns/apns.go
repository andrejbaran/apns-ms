// Example of a command line executable for running APNS Provider as "microservice" but despite it being an example it can be used out of the box for regular use.
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
// HTTP API
//
// API has 2 endpoints:
//
// * for sending raw push notifications (APN service).
//
// * for fetching expired device tokens (Feedback service).
//
// Note: sending push notification from template will be available soon.
//
// Raw push notification endpoint
//
// You can set URI for this endpoint by providing command line argument
//  --notification-endpoint="/my-send-push-notification-endpoint"
//
// This endpoint accepts POST requests with JSON formatted notification data. Notification data format resembles Apple's notification format specification and
// can be validated with following json schema:
//  {
//   "$schema":"http://json-schema.org/draft-04/schema#",
//   "id":"/",
//   "type":"object",
//   "additionalProperties":false,
//   "properties":{
//     "deviceToken":{
//       "id":"deviceToken",
//       "type":"string",
//       "pattern":"^(([a-f0-9]){2}){32}$",
//       "minLength":64,
//       "maxLength":64
//     },
//     "payload":{
//       "id":"payload",
//       "type":"object",
//       "additionalProperties":true,
//       "properties":{
//         "aps":{
//           "id":"aps",
//           "type":"object",
//           "additionalProperties":false,
//           "properties":{
//             "alert":{
//               "oneOf":[
//                 {
//                   "id":"alertObject",
//                   "type":"object",
//                   "additionalProperties":false,
//                   "properties":{
//                     "title":{
//                       "id":"title",
//                       "type":"string"
//                     },
//                     "body":{
//                       "id":"body",
//                       "type":"string"
//                     },
//                     "title-loc-key":{
//                       "id":"title-loc-key",
//                       "type":"string"
//                     },
//                     "title-loc-args":{
//                       "id":"title-loc-args",
//                       "type":"array",
//                       "minItems":0,
//                       "uniqueItems":false,
//                       "additionalItems":true,
//                       "items": {
//                         "type":"string"
//                       }
//                     },
//                     "action-loc-key":{
//                       "id":"action-loc-key",
//                       "type":"string"
//                     },
//                     "loc-key":{
//                       "id":"loc-key",
//                       "type":"string"
//                     },
//                     "loc-args":{
//                       "id":"loc-args",
//                       "type":"array",
//                       "minItems":0,
//                       "uniqueItems":true,
//                       "additionalItems":true,
//                       "items": {
//                         "type":"string"
//                       }
//                     },
//                     "launch-image":{
//                       "id":"launch-image",
//                       "type":"string"
//                     }
//                   }
//                 },
//                 {
//                  "id":"alertString",
//                  "type":"string"
//                 }
//               ]
//             },
//             "badge":{
//               "id":"badge",
//               "type":"integer",
//               "minimum": 0
//             },
//             "sound":{
//               "id":"sound",
//               "type":"string"
//             },
//             "category":{
//               "id":"category",
//               "type":"string"
//             },
//             "content-available":{
//               "id":"content-available",
//               "type":"integer"
//             }
//           },
//           "required":[
//             "alert"
//           ]
//         },
//         "customValues": {
//           "id":"aps",
//           "type":"object",
//           "additionalProperties":true
//         }
//       }
//     },
//     "identifier":{
//       "id":"identifier",
//       "type":"string"
//     },
//     "priority":{
//       "id":"priority",
//       "type":"integer",
//       "enum": [5, 10]
//     }
//   },
//   "required":[
//     "deviceToken",
//     "payload"
//   ]
//  }
//
// Possible responses:
//
// 	202 Accepted
// Means notification data is valid and notification was queued and will be send as soon as possible to APNS servers. Response content includes json encoded notification data.
// 	405 Method Not Allowed
// Means that request type was not "POST". Response Content-Length is zero.
// 	409 Conflict
// Means that notification data is not valid. Response content includes error message.
// 	503 Service Unavailable
// Means the processing queue is full and the request needs to be resend later. Response Content-Length is zero.
//
// Raw push notification endpoint example
//
// Request:
//  POST /my-send-push-notification-endpoint HTTP/1.1
//  Host: MY_APNS_MS_HOST:MY_APNS_MS_PORT
//  Content-Type: application/json
//
//  {
//     "deviceToken": "b8e0c9ce2114fc73adf117de0c97376626ef9c34bbfec4fe18e1fe0b96321cae",
//     "payload": {
//         "aps": {
//             "alert": "Hi there!",
//             "sound":"default"
//         },
//         "customValues": {
//             "weather":"It will be sunny today"
//         }
//     }
//  }
//
// Response:
//  HTTP/1.1 202 Accepted
//  Content-Type: application/json; charset=utf8
//  Content-Length: 199
//  Date: Wed, 21 Oct 2015 08:18:16 GMT
//
//  {
//   "deviceToken": "b8e0c9ce2114fc73adf117de0c97376626ef9c34bbfec4fe18e1fe0b96321cae",
//   "payload": {
//     "aps": {
//       "alert": "Hi there!",
//       "sound": "default"
//     },
//     "weather": "It will be sunny today"
//   },
//   "identifier": "0507e79b"
//  }
//
// Expired device tokens endpoint
//
// You can set URI for this endpoint by providing command line argument
//  --expired-devices-endpoint="/my-feedback-endpoint"
//
// This endpoint accepts GET requests. Response includes a json encoded list of expired device tokens if there where any new since the last check.
//
// Possible responses:
//
// 	200 OK
// Means that request to Feedback service was successfull. Response includes a json encoded list of expired device tokens with timestamp of the expiry.
// Per Apple's recommendation you should always check whether the device hasn't reregister after the timestamp of expiry. In that case the expiry should be ignored.
// 	405 Method Not Allowed
// Means that request type was not "GET". Response Content-Length is zero.
// 	500 Internal Server Error
// Means an error was encountered while processing of Feedback service response. Response include json encoded error message.
//
// Expired device tokens endpoint example
//
// Request:
// 	GET /my-feedback-endpoint HTTP/1.1
// 	Host: MY_APNS_MS_HOST:MY_APNS_MS_PORT
//
// Response:
//
//  HTTP/1.1 200 OK
//  Content-Type: application/json; charset=utf8
//  Content-Length: 199
//  Date: Wed, 21 Oct 2015 08:18:16 GMT
//
//  {
//   "devices": [
//     {
//       "timestamp": "2015-10-21T10:32:31+02:00",
//       "deviceToken": "b687baf21a5eb87c2977e113c0704b002067680f2101bbb4679fc366a9024fd4"
//     }
//   ]
//  }
//
//
//
//
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
