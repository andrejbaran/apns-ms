package apns

import (
	"bytes"
	"encoding/binary"
)

// SendNotificationCommandValue is the value of send push notification command in apns binary protocol
const SendNotificationCommandValue = 2

// PushNotificationCommand represents command for sending push notification
type PushNotificationCommand struct {
	Notification  *Notification
	errorsChannel chan CommandErrorInterface
}

// NewPushNotificationCommand creates a new send push notifiction command
func NewPushNotificationCommand(n *Notification) (cmd *PushNotificationCommand) {
	cmd = new(PushNotificationCommand)
	cmd.Notification = n
	cmd.errorsChannel = make(chan CommandErrorInterface)

	return
}

// Bytes returns send push notification command data
func (cmd *PushNotificationCommand) Bytes() ([]byte, error) {
	commandBuffer := &bytes.Buffer{}

	notificationBytes, err := cmd.Notification.Bytes()
	if err != nil {
		return nil, err
	}

	binary.Write(commandBuffer, binary.BigEndian, uint8(SendNotificationCommandValue))
	binary.Write(commandBuffer, binary.BigEndian, uint32(len(notificationBytes)))
	binary.Write(commandBuffer, binary.BigEndian, notificationBytes)

	cmdBytes := commandBuffer.Bytes()

	return cmdBytes, nil
}

// Data returns data associated with command, in this case the Notification struct
func (cmd *PushNotificationCommand) Data() interface{} {
	return cmd.Notification
}

// Identifier returns command identifier (in this case notification identifier)
func (cmd *PushNotificationCommand) Identifier() string {
	identifier := ""

	if cmd.Notification != nil {
		identifier = cmd.Notification.NotificationIdentifier
	}

	return identifier
}

// String returns a human readable description of the command
func (cmd *PushNotificationCommand) String() string {
	return "Push Notification #" + cmd.Identifier()
}

// Errors returns a channel to which errors will be sent
func (cmd *PushNotificationCommand) Errors() chan CommandErrorInterface {
	return cmd.errorsChannel
}
