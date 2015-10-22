package apns

import (
	// "errors"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestNewNotification(t *testing.T) {
	n := NewNotification()
	assert := assert.New(t)

	assert.NotEqual(nil, n.Payload, "Notification payload shouldn't be nil")
	assert.NotEqual(nil, n.Payload.Aps, "Notification payload 'aps' shouldn't be nil")

	assert.Len(n.NotificationIdentifier, 8, "Generated notification identifier should be 8 bytes")
}

func BenchmarkGenerateNotificationIdentifier(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewNotification()
	}
}

func TestNotificationDeviceTokenValidation(t *testing.T) {
	n := NewNotification()

	assert := assert.New(t)

	var referenceError string
	var notificationError error

	// hex encoded
	n.DeviceToken = "Some device token"
	referenceError = "Device token should be hex encoded " + strconv.Itoa(DeviceTokenItemLength) + " bytes long binary string"
	_, notificationError = n.Bytes()
	assert.Contains(notificationError.Error(), referenceError, "Invalid device token error message")

	// length
	n.DeviceToken = "000000000000000000000000000000000000000000000000000000000000"
	referenceError = "Device token length is 30 bytes but should be " + strconv.Itoa(DeviceTokenItemLength) + " bytes"
	_, notificationError = n.Bytes()
	assert.Contains(notificationError.Error(), referenceError, "Invalid device token error message")
}

func TestNotificationIdentifierValidation(t *testing.T) {
	n := NewNotification()
	n.DeviceToken = "0000000000000000000000000000000000000000000000000000000000000000"

	assert := assert.New(t)

	var referenceError string
	var notificationError error

	// hex encoded
	n.NotificationIdentifier = "An identifier"
	referenceError = "Notification identifier should be hex encoded " + strconv.Itoa(NotificationIdentifierItemLength) + " bytes long binary string"
	_, notificationError = n.Bytes()
	assert.Contains(notificationError.Error(), referenceError, "Invalid notification identifier error message")

	// length
	n.NotificationIdentifier = "aabbccddee"
	referenceError = "Notification identifier length is 5 bytes but should be " + strconv.Itoa(NotificationIdentifierItemLength) + " bytes"
	_, notificationError = n.Bytes()
	assert.Contains(notificationError.Error(), referenceError, "Invalid notification identifier error message")
}

func TestNotificationPayloadValidation(t *testing.T) {
	n := NewNotification()
	n.DeviceToken = "0000000000000000000000000000000000000000000000000000000000000000"

	assert := assert.New(t)

	var referenceError string
	var notificationError error

	// length
	alert := new(Alert)
	alert.Body = ""
	for i := 0; i < 2048; i++ {
		alert.Body += "0"
	}
	n.Payload.Aps.Alert = alert
	referenceError = "Notification payload size is 2077 bytes but should be " + strconv.Itoa(PayloadItemMaxLength) + " bytes at maximum"
	_, notificationError = n.Bytes()
	assert.Contains(notificationError.Error(), referenceError, "Invalid notification payload error message")
}

func TestNotificationPayloadMarshalling(t *testing.T) {
	n := NewNotification()
	n.NotificationIdentifier = "aabbccdd"
	n.DeviceToken = "0000000000000000000000000000000000000000000000000000000000000000"

	assert := assert.New(t)

	var referenceJSONString string
	var notificationJSONString string
	var notificationError error

	// format
	alert := new(Alert)
	alert.Title = "Hi!"
	alert.Body = "Hello World :)"
	alert.ActionLocalizationKey = "_THE_ACTION_"
	alert.BodyLocalizationArgs = append(alert.BodyLocalizationArgs, "ARG1")
	alert.BodyLocalizationArgs = append(alert.BodyLocalizationArgs, "ARG2")
	alert.BodyLocalizationArgs = append(alert.BodyLocalizationArgs, "ARG3")
	alert.BodyLocalizationArgs = append(alert.BodyLocalizationArgs, "ARG4")
	alert.BodyLocalizationKey = "_THE_BODY_"
	alert.LaunchImage = "image.png"
	alert.TitleLocalizationKey = "_THE_TITLE_"
	alert.TitleLocalizationdArgs = append(alert.TitleLocalizationdArgs, "ARG1")

	n.Payload.AddCustomField("abc", "def")
	n.Payload.Aps.Alert = alert
	n.Payload.Aps.Sound = "default"
	n.Payload.Aps.Badge = 123
	n.Payload.Aps.Category = "category"
	n.Payload.Aps.ContentAvailable = 1

	referenceJSONString = "{\"abc\":\"def\",\"aps\":{\"alert\":{\"title\":\"Hi!\",\"body\":\"Hello World :)\",\"title-loc-key\":\"_THE_TITLE_\",\"title-loc-args\":[\"ARG1\"],\"action-loc-key\":\"_THE_ACTION_\",\"loc-key\":\"_THE_BODY_\",\"loc-args\":[\"ARG1\",\"ARG2\",\"ARG3\",\"ARG4\"],\"launch-image\":\"image.png\"},\"badge\":123,\"sound\":\"default\",\"content-available\":1,\"category\":\"category\"}}"
	notificationJSONString, notificationError = n.Payload.JSONString()

	assert.Nil(notificationError, "Marshalling shouldn't produce error")
	assert.Contains(notificationJSONString, referenceJSONString, "JSON string should be equal")
}
