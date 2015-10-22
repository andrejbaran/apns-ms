package apns

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/mitchellh/mapstructure"
	"strconv"
	"time"
)

const (
	// DeviceTokenItemID is the ID of device token item in apns binary protocol
	DeviceTokenItemID = 1
	// DeviceTokenItemLength is the length of the device token item
	DeviceTokenItemLength = 32
	// PayloadItemID is the ID of payload item in apns binary protocol
	PayloadItemID = 2
	// PayloadItemMaxLength is the maximum length of the payload item
	PayloadItemMaxLength = 2048
	// NotificationIdentifierItemID is the ID of notification identifier item in apns binary protocol
	NotificationIdentifierItemID = 3
	// NotificationIdentifierItemLength is the length of notification identifier item
	NotificationIdentifierItemLength = 4
	// ExpirationDateItemID is the ID of expiration date item in apns binary protocol
	ExpirationDateItemID = 4
	// ExpirationDateItemLength is the length of expiration date item
	ExpirationDateItemLength = 4
	// PriorityItemID is the ID of priority item in apns binary protocol
	PriorityItemID = 5
	// PriorityItemLength is the length of priority item
	PriorityItemLength = 1
)

// Alert struct represents alert dictionary (https://developer.apple.com/library/prerelease/watchos/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/ApplePushService.html#//apple_ref/doc/uid/TP40008194-CH100-SW20)
type Alert struct {
	Title                  string   `json:"title,omitempty", mapstructure:"title"`
	Body                   string   `json:"body,omitempty", mapstructure:"body"`
	TitleLocalizationKey   string   `json:"title-loc-key,omitempty", mapstructure:"title-loc-key"`
	TitleLocalizationdArgs []string `json:"title-loc-args,omitempty", mapstructure:"title-loc-args"`
	ActionLocalizationKey  string   `json:"action-loc-key,omitempty", mapstructure:"action-loc-key"`
	BodyLocalizationKey    string   `json:"loc-key,omitempty", mapstructure:"loc-key"`
	BodyLocalizationArgs   []string `json:"loc-args,omitempty", mapstructure:"loc-args"`
	LaunchImage            string   `json:"launch-image,omitempty", mapstructure:"launch-image"`
}

// Aps struct represents aps dictionary (https://developer.apple.com/library/prerelease/watchos/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/ApplePushService.html#//apple_ref/doc/uid/TP40008194-CH100-SW2)
type Aps struct {
	Alert            interface{} `json:"alert,omitempty"`
	Badge            int         `json:"badge,omitempty"`
	Sound            string      `json:"sound,omitempty"`
	ContentAvailable int         `json:"content-available,omitempty"`
	Category         string      `json:"category,omitempty"`
}

// NewAps creates a new blank notification payload aps object
func NewAps() *Aps {
	aps := new(Aps)
	return aps
}

// Payload struct represents the whole notification payload (https://developer.apple.com/library/prerelease/watchos/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/ApplePushService.html#//apple_ref/doc/uid/TP40008194-CH100-SW1)
type Payload struct {
	Aps          *Aps `json:"aps,omitempty"`
	customValues map[string]interface{}
}

// NewPayload creates a new blank notification payload object
func NewPayload() *Payload {
	payload := new(Payload)
	payload.Aps = NewAps()

	return payload
}

// AddCustomField adds custom field to notification payload
func (p *Payload) AddCustomField(key string, value interface{}) {
	if p.customValues == nil {
		p.customValues = make(map[string]interface{})
	}

	p.customValues[key] = value
}

// MarshalJSON implements custom marshalling of notification payload to json
func (p *Payload) MarshalJSON() (jsonBytes []byte, err error) {
	payload := make(map[string]interface{})

	if p.Aps == nil {
		err = errors.New("apns/notification: 'aps' object is required")
		return
	}

	payload["aps"] = p.Aps

	for key, value := range p.customValues {
		if key == "aps" {
			jsonBytes = nil
			err = errors.New("apns/notification: 'aps' is a reserved and cannot be used for custom field")
			return
		}
		payload[key] = value
	}

	jsonBytes, err = json.Marshal(payload)

	return
}

// JSON returns payload data marshalled into JSON
func (p *Payload) JSON() ([]byte, error) {
	return json.Marshal(p)
}

// JSONString returns payload data as JSON string
func (p *Payload) JSONString() (string, error) {
	json, err := p.JSON()
	return string(json), err
}

// Notification struct represents push notification
type Notification struct {
	DeviceToken            string     `json:"deviceToken,omitempty"`
	Payload                *Payload   `json:"payload,omitempty"`
	NotificationIdentifier string     `json:"identifier,omitempty"`
	ExpirationDate         *time.Time `json:"expires,omitempty"`
	Priority               uint8      `json:"priority,omitempty"`
}

// NewNotification creates a new blank notification object
func NewNotification() *Notification {
	var randomID []byte

	notification := new(Notification)
	notification.Payload = NewPayload()

	randomID = make([]byte, 4)

	_, err := rand.Read(randomID)
	if err == nil {
		// checksum := crc32.ChecksumIEEE(randomID)
		// buffer := new(bytes.Buffer)

		// binary.Write(buffer, binary.BigEndian, checksum)
		notification.NotificationIdentifier = hex.EncodeToString(randomID)
	}

	return notification
}

// UnmarshalJSON implements custom marshalling of notification json
func (n *Notification) UnmarshalJSON(data []byte) (err error) {
	type NotificationAlias Notification

	type fakePayload struct {
		Aps          *Aps                   `json:"aps"`
		CustomValues map[string]interface{} `json:"customValues"`
	}

	var fakeNotification = &struct {
		Payload *fakePayload `json:"payload,omitempty"`
		*NotificationAlias
	}{
		NotificationAlias: (*NotificationAlias)(&Notification{}),
	}

	fakeNotification.Payload = &fakePayload{}

	err = json.Unmarshal(data, fakeNotification)
	if err != nil {
		return
	}

	n.DeviceToken = fakeNotification.DeviceToken

	// set provided notification identifier otherwise keep generated one
	if fakeNotification.NotificationIdentifier != "" {
		n.NotificationIdentifier = fakeNotification.NotificationIdentifier
	}
	n.ExpirationDate = fakeNotification.ExpirationDate
	n.Priority = fakeNotification.Priority

	n.Payload = NewPayload()
	n.Payload.customValues = fakeNotification.Payload.CustomValues

	if fakeNotification.Payload.Aps != nil {
		_, alertIsString := fakeNotification.Payload.Aps.Alert.(string)

		if alertIsString {
			n.Payload.Aps = fakeNotification.Payload.Aps
		} else {
			alertDictionary := new(Alert)
			decodeError := mapstructure.Decode(fakeNotification.Payload.Aps.Alert, &alertDictionary)

			if decodeError != nil {
				logger.Debugf("apns/notification: Error occured during decoding alert dictionary %+v", fakeNotification.Payload.Aps.Alert)
				err = errors.New("apns/notification: Invalid alert dictionary format")
				return
			}

			n.Payload.Aps = fakeNotification.Payload.Aps
			n.Payload.Aps.Alert = alertDictionary
		}
	}

	return nil
}

// Bytes returns binary representation of send push notification (https://developer.apple.com/library/prerelease/watchos/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/CommunicatingWIthAPS.html#//apple_ref/doc/uid/TP40008194-CH101-SW4)
func (n *Notification) Bytes() ([]byte, error) {
	frameBuffer := &bytes.Buffer{}

	// Device token
	token, deviceTokenError := hex.DecodeString(n.DeviceToken)
	if deviceTokenError != nil {
		return nil, deviceTokenError
	}
	if len(token) == DeviceTokenItemLength {
		binary.Write(frameBuffer, binary.BigEndian, uint8(DeviceTokenItemID))
		binary.Write(frameBuffer, binary.BigEndian, uint16(DeviceTokenItemLength))
		binary.Write(frameBuffer, binary.BigEndian, token)
	} else {
		return nil, errors.New("apns/notification: Device token has to be hex encoded " + strconv.Itoa(DeviceTokenItemLength) + " bytes long binary string")
	}

	// Payload
	payload, payloadError := n.Payload.JSON()
	if payloadError != nil {
		return nil, payloadError
	}
	if len(payload) > PayloadItemMaxLength {
		return nil, errors.New("apns/notification: Notification payload size has to be " + strconv.Itoa(PayloadItemMaxLength) + " bytes at maximum")
	}
	binary.Write(frameBuffer, binary.BigEndian, uint8(PayloadItemID))
	binary.Write(frameBuffer, binary.BigEndian, uint16(len(payload)))
	binary.Write(frameBuffer, binary.BigEndian, payload)

	// Notification Identifer
	identifier, identifierError := hex.DecodeString(n.NotificationIdentifier)
	if identifierError != nil {
		return nil, identifierError
	}
	if len(identifier) != NotificationIdentifierItemLength {
		return nil, errors.New("apns/notification: Notification identifier has to be a hex encoded " + strconv.Itoa(NotificationIdentifierItemLength) + " bytes longs binary string")
	}
	binary.Write(frameBuffer, binary.BigEndian, uint8(NotificationIdentifierItemID))
	binary.Write(frameBuffer, binary.BigEndian, uint16(NotificationIdentifierItemLength))
	binary.Write(frameBuffer, binary.BigEndian, identifier)

	// Expiration Date
	if n.ExpirationDate != nil {
		binary.Write(frameBuffer, binary.BigEndian, uint8(ExpirationDateItemID))
		binary.Write(frameBuffer, binary.BigEndian, uint16(ExpirationDateItemLength))
		binary.Write(frameBuffer, binary.BigEndian, n.ExpirationDate.Unix())
	}

	// Priority
	binary.Write(frameBuffer, binary.BigEndian, uint8(PriorityItemID))
	binary.Write(frameBuffer, binary.BigEndian, uint16(PriorityItemLength))
	binary.Write(frameBuffer, binary.BigEndian, n.Priority)

	return frameBuffer.Bytes(), nil
}
