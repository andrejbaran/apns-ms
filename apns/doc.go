// Package apns exposes APNS Provider for Apple's APNS and Feedback services and uses job/worker pattern to process notifications concurrently.
// Each worker establishes it's own TLS connection to APNS gateway. When an error response is received from APNS server, worker tries to reconnect automatically.
package apns
