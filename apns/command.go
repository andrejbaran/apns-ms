package apns

import (
	"encoding/hex"
	"errors"
)

// CommandInterface specifies an interface for APNS commands
type CommandInterface interface {
	Identifier() string
	Bytes() ([]byte, error)
	Data() interface{}
	String() string
	Errors() chan CommandErrorInterface
}

// CommandErrorInterface specifies and interface for command execution errors
type CommandErrorInterface interface {
	Error() string
	GetError() error
	GetCommand() CommandInterface
}

// CommandError is a generic command error
type CommandError struct {
	commandError error
	command      CommandInterface
}

///
///
/// Push Notification Command Error
///
///

// PushNotificationErrorStatuses represents APNS error status codes (https://developer.apple.com/library/ios/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/CommunicatingWIthAPS.html#//apple_ref/doc/uid/TP40008194-CH101-SW12)
var PushNotificationErrorStatuses = map[uint8]string{
	0:   "No errors encountered",
	1:   "Processing error",
	2:   "Missing device token",
	3:   "Missing topic",
	4:   "Missing payload",
	5:   "Invalid token size",
	6:   "Invalid topic size",
	7:   "Invalid payload size",
	8:   "Invalid token",
	10:  "Shutdown",
	255: "Unknown",
}

// NewCommandError creates and returns new generic command execution error
func NewCommandError(err error, cmd CommandInterface) *CommandError {
	genericError := new(CommandError)
	genericError.command = cmd
	genericError.commandError = err

	return genericError
}

// NewCommandErrorFromAPNSResponse creates and returns error representing APNS response
func NewCommandErrorFromAPNSResponse(data []byte, cmd CommandInterface) (commandError *CommandError) {
	var err error

	if len(data) != 6 {
		err = errors.New("apns: Unrecognized APNS response")
	} else {
		statusCode := uint8(data[1])
		notificationIdentifier := hex.EncodeToString(data[2:])

		if apnsErrorDescription := PushNotificationErrorStatuses[statusCode]; apnsErrorDescription != "" {
			err = errors.New("apns: " + apnsErrorDescription + " for notification #" + notificationIdentifier)
		}
	}

	commandError = NewCommandError(err, cmd)
	return
}

// Error implements standard go error interface
func (ge *CommandError) Error() string {
	if ge != nil && ge.commandError != nil {
		return ge.commandError.Error()
	}

	return ""
}

// GetError returns underlying error object
func (ge *CommandError) GetError() error {
	if ge != nil && ge.commandError != nil {
		return ge.commandError
	}

	return nil
}

// GetCommand returns command this error belongs to
func (ge *CommandError) GetCommand() CommandInterface {
	return ge.command
}
