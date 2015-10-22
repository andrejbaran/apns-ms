# APNS Provider as "micro-service"

This package consists of two main sub-packages `apns` and `server`. There is also `apns` binary which can be seen as an example of "micro-service" implementation but is ready for use out of the box.

`apns` package exposes APNS Provider for Apple's APNS and Feedback services. It uses job/worker pattern to process notifications concurrently. Each worker establishes it's own TLS connection to APNS gateway. When an error response is received from APNS server, worker tries to reconnect automatically.

`server` package exposes HTTP API in form of handler functions ready for use with go's `net/http` package but can also be used with other http servers like `falcore`.

## Usage
#### Installation

```bash
go get github.com/andrejbaran/apns-microservice
```

#### Logging

Both packages can be provided a logger implementing `apns.LoggerInterface` interface, which is based on CoreOS package [capnslog](https://github.com/coreos/pkg/tree/master/capnslog) (used by `apns` binary). For the specifics of the `apns.LoggerInterface` see the documentation.

It should be fairly simple to create a wrapper for your favorite logger so it implements implement `apns.LoggerInterface`.

By default both packages don't log anything until you set the logger.

#### Command line flags
Both packages specify their command line flags that can be used if there's need for them.

`apns` flags and their defaults:
```
--apns-gate-port=2195: Apple's APNS port number
--apns-gate-production="gateway.push.apple.com": FQDN of Apple's APNS production gateway.
--apns-gate-sandbox="gateway.sandbox.push.apple.com": FQDN of Apple's APNS sandbox gateway.
--cert="": Absolute path to certificate file. Certificate is expected be in PEM format.
--cert-key="": Absolute path to certificate private key file. Certificate key is expected be in PEM format.
--env="sandbox": Environment of Apple's APNS and Feedback service gateways. For production use specify "production", for testing specify "sandbox".
--feedback-gate-port=2196: Apple's Feedback service port number
--feedback-gate-production="feedback.push.apple.com": FQDN of Apple's Feedback service production gateway.
--feedback-gate-sandbox="feedback.sandbox.push.apple.com": FQDN of Apple's Feedback service sandbox gateway.
--max-notifications=100000: Number of notification that can be queued for processing at once. Once the queue is full all requests to raw push notification endpoint will result in 503 Service Unavailable response.
--workers=4: Number of workers that concurently process push notifications. Defaults to 2 * Number of CPU cores.
```

`server` flags and their defaults:
```
--address=0.0.0.0: IP address the HTTP server should bind to.
--expired-devices-endpoint="/expired-devices": URI of Expired device tokens endpoint.
--notification-endpoint="/notification": URI of Raw push notification endpoint.
--port=9090: Port on which HTTP should listen on.
```

#### Example usage of `apns` package as library

```go
import "github.com/andrejbaran/apns-ms/apns"

func main() {
    // use command line flags if you want to. just make sure to call SetupCommandLineFlags and parse them before you create new client config
    apns.SetupCommandLineFlags(pflag.CommandLine)
	pflag.Parse()

	config := apns.NewClientConfig()

    // or set config manually
    config.NumberOfWorkers = 10
    config.Env = "production"

    // create the client
	client, err := apns.NewClient(config)
	if err != nil {
		fmt.Printf("Couldn't create client because: %s", err)
	}

    // create notification
    notification := apns.NewNotification()
    notification.DeviceToken = "32 byte hex encoded binary string"
    notification.Payload.Aps.Alert = "Hi there!"

    // create a push notification command
    cmd := apns.NewPushNotificationCommand(notification)

    // execute the command (queue for execution/processing)
    queueError := client.ExecuteCommand(cmd)

    // queueError is not nil only if the command queue is full
    if queueError != nil {
        fmt.Print("Command couldn't be processed because the command queue is full. Try again little later.")
    }

    // listen for command errors
    commandError := <- cmd.Errors()
    if commandError != nil {
        fmt.Printf("Command failed because: %s", commandError)
    }
}
```

#### Using `apns` binary

Print usage (prints all available command line flags):
```bash
apns --help
```
`apns` binary logs to stdout.

## HTTP API

Currently there are 2 endpoints:
 * for sending raw push notifications (APN service).
 * for fetching expired device tokens (Feedback service).

Note: sending push notification from templates is on the roadmap.

### Raw push notification endpoint

You can set URI for this endpoint by providing command line argument `--notification-endpoint`

This endpoint accepts POST requests with JSON formatted notification data. Notification data format resembles Apple's notification format specification and can be validated with following json schema:
```json
  {
   "$schema":"http:json-schema.org/draft-04/schema#",
   "id":"/",
   "type":"object",
   "additionalProperties":false,
   "properties":{
     "deviceToken":{
       "id":"deviceToken",
       "type":"string",
       "pattern":"^(([a-f0-9]){2}){32}$",
       "minLength":64,
       "maxLength":64
     },
     "payload":{
       "id":"payload",
       "type":"object",
       "additionalProperties":true,
       "properties":{
         "aps":{
           "id":"aps",
           "type":"object",
           "additionalProperties":false,
           "properties":{
             "alert":{
               "oneOf":[
                 {
                   "id":"alertObject",
                   "type":"object",
                   "additionalProperties":false,
                   "properties":{
                     "title":{
                       "id":"title",
                       "type":"string"
                     },
                     "body":{
                       "id":"body",
                       "type":"string"
                     },
                     "title-loc-key":{
                       "id":"title-loc-key",
                       "type":"string"
                     },
                     "title-loc-args":{
                       "id":"title-loc-args",
                       "type":"array",
                       "minItems":0,
                       "uniqueItems":false,
                       "additionalItems":true,
                       "items": {
                         "type":"string"
                       }
                     },
                     "action-loc-key":{
                       "id":"action-loc-key",
                       "type":"string"
                     },
                     "loc-key":{
                       "id":"loc-key",
                       "type":"string"
                     },
                     "loc-args":{
                       "id":"loc-args",
                       "type":"array",
                       "minItems":0,
                       "uniqueItems":true,
                       "additionalItems":true,
                       "items": {
                         "type":"string"
                       }
                     },
                     "launch-image":{
                       "id":"launch-image",
                       "type":"string"
                     }
                   }
                 },
                 {
                  "id":"alertString",
                  "type":"string"
                 }
               ]
             },
             "badge":{
               "id":"badge",
               "type":"integer",
               "minimum": 0
             },
             "sound":{
               "id":"sound",
               "type":"string"
             },
             "category":{
               "id":"category",
               "type":"string"
             },
             "content-available":{
               "id":"content-available",
               "type":"integer"
             }
           },
           "required":[
             "alert"
           ]
         },
         "customValues": {
           "id":"aps",
           "type":"object",
           "additionalProperties":true
         }
       }
     },
     "identifier":{
       "id":"identifier",
       "type":"string"
     },
     "priority":{
       "id":"priority",
       "type":"integer",
       "enum": [5, 10]
     }
   },
   "required":[
     "deviceToken",
     "payload"
   ]
  }
```

#### Possible responses:

`202 Accepted`
> Means notification data is valid and notification was queued and will be send as soon as possible to APNS servers. Response content includes json encoded notification data.

`405 Method Not Allowed`
> Means that request type was not "POST". Response Content-Length is zero.

`409 Conflict`
> Means that notification data is not valid. Response content includes error message.

`503 Service Unavailable`
> Means the processing queue is full and the request needs to be resend later. Response Content-Length is zero.

#### Raw push notification endpoint example

##### Request
```HTTP
POST /my-send-push-notification-endpoint HTTP/1.1
Host: MY_APNS_MS_HOST:MY_APNS_MS_PORT
Content-Type: application/json

{
    "deviceToken": "b8e0c9ce2114fc73adf117de0c97376626ef9c34bbfec4fe18e1fe0b96321cae",
    "payload": {
        "aps": {
            "alert": "Hi there!",
            "sound":"default"
        },
        "customValues": {
            "weather":"It will be sunny today"
        }
    }
}
```

##### Response
```HTTP
HTTP/1.1 202 Accepted
Content-Type: application/json; charset=utf8
Content-Length: 199
Date: Wed, 21 Oct 2015 08:18:16 GMT

{
    "deviceToken": "b8e0c9ce2114fc73adf117de0c97376626ef9c34bbfec4fe18e1fe0b96321cae",
    "payload": {
        "aps": {
            "alert": "Hi there!",
            "sound": "default"
        },
        "weather": "It will be sunny today"
    },
    "identifier": "0507e79b"
}
```

### Expired device tokens endpoint

You can set URI for this endpoint by providing command line argument `--expired-devices-endpoint`

This endpoint accepts GET requests. Response includes a json encoded list of expired device tokens if there where any new since the last check.

#### Possible responses:

`200 OK`
> Means that request to Feedback service was successfull. Response includes a json encoded list of expired device tokens with timestamp of the expiry.
Per Apple's recommendation you should always check whether the device hasn't reregister after the timestamp of expiry. In that case the expiry should be ignored.

`405 Method Not Allowed`
> Means that request type was not "GET". Response Content-Length is zero.

`500 Internal Server Error`
> Means an error was encountered while processing of Feedback service response. Response include json encoded error message.

#### Expired device tokens endpoint example

##### Request:
```http
GET /my-feedback-endpoint HTTP/1.1
Host: MY_APNS_MS_HOST:MY_APNS_MS_PORT
```

##### Response:
```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf8
Content-Length: 199
Date: Wed, 21 Oct 2015 08:18:16 GMT

{
    "devices": [
     {
       "timestamp": "2015-10-21T10:32:31+02:00",
       "deviceToken": "b687baf21a5eb87c2977e113c0704b002067680f2101bbb4679fc366a9024fd4"
     }
    ]
}
```

## Docs
godoc.org

## Roadmap
- Tests!!!
- JSON Schema validation
- Improve error handling
- Improve docs
- Automatic notification priority
- Templates for notifications (to make use of [ notification actions/categories](https://developer.apple.com/library/ios/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/IPhoneOSClientImp.html#//apple_ref/doc/uid/TP40008194-CH103-SW26) easier)
- Adaptive number of workers (depending on amount of requests)
- Stats

## License
MIT
