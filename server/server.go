package server

import (
	"apns-ms/apns"
	"encoding/json"
	"errors"
	"github.com/spf13/pflag"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	// Address is IP address the HTTP server should bind to
	Address = net.ParseIP("0.0.0.0")
	// Port is port on which HTTP server is listening
	Port uint16 = 9090
	// RawNotificationEndpoint is URI of Raw push notification endpoint
	RawNotificationEndpoint = "/notification"
	// ExpiredDeviceTokensEndpoint is URI of Expired device tokens endpoint
	ExpiredDeviceTokensEndpoint = "/expired-devices"

	notificationCounter uint64
	feedbackCounter     uint64
)

func setupHTTPCommandLineFlags(fs *pflag.FlagSet) {
	fs.IPVar(&Address, "address", Address, "IP address the HTTP server should bind to.")
	fs.Uint16Var(&Port, "port", Port, "Port on which HTTP server should listen on.")
	fs.StringVar(&RawNotificationEndpoint, "notification-endpoint", RawNotificationEndpoint, "URI of Raw push notification endpoint.")
	fs.StringVar(&ExpiredDeviceTokensEndpoint, "expired-devices-endpoint", ExpiredDeviceTokensEndpoint, "URI of Expired device tokens endpoint.")
}

// NewRawNotificationHTTPHandlerFunc returns a net/http compatible request handler function that expects raw notification data and sends notification to APN service
func NewRawNotificationHTTPHandlerFunc(c *apns.Client) (f http.HandlerFunc) {
	f = func(c *apns.Client) http.HandlerFunc {
		var handlerFunc http.HandlerFunc

		handlerFunc = func(w http.ResponseWriter, req *http.Request) {
			startTime := time.Now()

			atomic.AddUint64(&notificationCounter, 1)

			var responseData []byte

			logger.Infof("Received send push notification request #%d", notificationCounter)

			responseHeaders := w.Header()
			responseHeaders.Set("Content-Type", "application/json; charset=utf8")

			// check method
			if req.Method != "POST" {
				defer finishResponse("Send push notification", notificationCounter, w, http.StatusMethodNotAllowed, responseData, startTime)
				return
			}

			// read body data
			bodyDecoder := json.NewDecoder(req.Body)

			notification := apns.NewNotification()
			bodyError := bodyDecoder.Decode(notification)

			if bodyError != nil {
				if bodyError == io.EOF {
					bodyError = errors.New("Notification data is missing")
				}

				logger.Errorf("Error occured during processing of notification data: %+v", bodyError)

				responseData, _ = json.Marshal(&struct {
					Error string `json:"error"`
				}{
					Error: bodyError.Error(),
				})

				defer finishResponse("Send push notification", notificationCounter, w, http.StatusConflict, responseData, startTime)
				return
			}

			cmd := apns.NewPushNotificationCommand(notification)
			err := c.ExecuteCommand(cmd)

			commandError := <-cmd.Errors()

			if commandError != nil {
				logger.Debugf("Command error: %s", commandError.Error())
			}

			if err != nil {
				responseData, _ = json.Marshal(&struct {
					Error string `json:"error"`
				}{
					Error: err.Error(),
				})

				defer finishResponse("Send push notification", notificationCounter, w, http.StatusServiceUnavailable, responseData, startTime)
				return
			}

			if commandError != nil {
				responseData, _ = json.Marshal(&struct {
					Error string `json:"error"`
				}{
					Error: commandError.Error(),
				})

				defer finishResponse("Send push notification", notificationCounter, w, http.StatusConflict, responseData, startTime)
				return
			}

			responseData, _ = json.Marshal(notification)

			finishResponse("Send push notification", notificationCounter, w, http.StatusAccepted, responseData, startTime)
		}

		return handlerFunc
	}(c)

	return
}

// NewExpiredDevicesHTTPHandlerFunc returns a net/http compatible request handler function for fetching Feedback service data
func NewExpiredDevicesHTTPHandlerFunc(c *apns.Client) (f http.HandlerFunc) {
	f = func(c *apns.Client) http.HandlerFunc {
		var handlerFunc http.HandlerFunc

		handlerFunc = func(w http.ResponseWriter, req *http.Request) {
			startTime := time.Now()

			atomic.AddUint64(&feedbackCounter, 1)

			var responseData []byte

			logger.Infof("Received check feedback service request #%d", feedbackCounter)

			responseHeaders := w.Header()
			responseHeaders.Set("Content-Type", "application/json; charset=utf8")

			// check method
			if req.Method != "GET" {
				defer finishResponse("Check feedback service", feedbackCounter, w, http.StatusMethodNotAllowed, responseData, startTime)
				return
			}

			response, err := c.CheckFeedbackService()

			if err != nil {
				responseData, _ = json.Marshal(&struct {
					Error string `json:"error"`
				}{
					Error: err.Error(),
				})

				defer finishResponse("Check feedback service", feedbackCounter, w, http.StatusInternalServerError, responseData, startTime)
				return
			}

			responseData, _ = json.Marshal(response)

			finishResponse("Check feedback service", feedbackCounter, w, http.StatusOK, responseData, startTime)
		}

		return handlerFunc
	}(c)

	return
}

func finishResponse(requestType string, counter uint64, w http.ResponseWriter, responseStatus int, responseData []byte, startTime time.Time) {
	w.WriteHeader(responseStatus)

	if len(responseData) > 0 {
		w.Write(responseData)
	}

	endTime := time.Now()
	logger.Infof("%s request #%d finished with %s (%d) in %s", requestType, counter, http.StatusText(responseStatus), responseStatus, endTime.Sub(startTime))
}
