package notification

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
)

// Client describes the functionality that shoult be provided by notification service clients
type Client interface {
	RegisterDeviceForPush(*DeviceRegistrationInfo) error
	SendNotification(*Notification) error
}

// ClientConfig represents the config aspects used by the notifications client
type ClientConfig struct {
	SQSDeviceRegistrationURL string
	SQSNotificationURL       string
	Session                  *session.Session
}

type client struct {
	sqsAPI sqsiface.SQSAPI
	config *ClientConfig
}

// NewClient returns an initialized instance of client
func NewClient(config *ClientConfig) Client {
	return &client{
		config: config,
		sqsAPI: sqs.New(config.Session),
	}
}

func (c *client) RegisterDeviceForPush(dri *DeviceRegistrationInfo) error {
	return errors.Trace(c.sendSQSMessage(dri, c.config.SQSDeviceRegistrationURL))
}

func (c *client) SendNotification(n *Notification) error {
	return errors.Trace(c.sendSQSMessage(n, c.config.SQSNotificationURL))
}

func (c *client) sendSQSMessage(message interface{}, queueURL string) error {
	body, err := json.Marshal(message)
	if err != nil {
		return errors.Trace(err)
	}
	_, err = c.sqsAPI.SendMessage(&sqs.SendMessageInput{
		MessageBody: ptr.String(string(body)),
		QueueUrl:    ptr.String(queueURL),
	})
	return errors.Trace(err)
}
