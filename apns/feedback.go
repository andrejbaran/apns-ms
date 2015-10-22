package apns

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"
)

const (
	// TimestampItemLength is the length of timestamp item
	TimestampItemLength = 4
	// DeviceTokenLengthItemLength is the length of length of device token item
	DeviceTokenLengthItemLength = 2
)

// FeedbackResponse holds all device entries from feedback service response
type FeedbackResponse struct {
	Devices []*FeedbackDeviceEntry `json:"devices"`
}

// NewFeedbackResponse returns a new feedback tuple object
func NewFeedbackResponse() *FeedbackResponse {
	response := new(FeedbackResponse)
	response.Devices = make([]*FeedbackDeviceEntry, 0)
	return response
}

// FeedbackDeviceEntry struct represents feedback tuple (https://developer.apple.com/library/ios/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/CommunicatingWIthAPS.html#//apple_ref/doc/uid/TP40008194-CH101-SW5)
type FeedbackDeviceEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	DeviceToken string    `json:"deviceToken"`
}

// NewFeedbackDeviceEntry returns a new feedback tuple object
func NewFeedbackDeviceEntry() *FeedbackDeviceEntry {
	entry := new(FeedbackDeviceEntry)
	return entry
}

func (fs *FeedbackResponse) addEntryFromBytes(data []byte) (err error) {
	err = nil

	if len(data) != TimestampItemLength+DeviceTokenLengthItemLength+DeviceTokenItemLength {
		err = errors.New("apns: Unrecognized Feedback Service entry")
		return
	}

	var timestamp uint32
	timestampBuffer := bytes.NewBuffer(data[0:4])
	err = binary.Read(timestampBuffer, binary.BigEndian, &timestamp)
	if err != nil {
		return
	}

	entry := NewFeedbackDeviceEntry()
	entry.Timestamp = time.Unix(int64(timestamp), 0)
	entry.DeviceToken = hex.EncodeToString(data[6:])

	fs.Devices = append(fs.Devices, entry)

	return
}
