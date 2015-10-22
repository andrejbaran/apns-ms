// Package server exposes HTTP API in form of handler functions ready for use with go's `net/http` package but can also be used with other http servers like `falcore`.
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
package server
