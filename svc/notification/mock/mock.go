package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/notification"
)

// Compile time check to make sure the mock conforms to the interface
var _ notification.Client = &Client{}

// Client is a mock for the directory service client.
type Client struct {
	*mock.Expector
}

// New returns an initialized Client.
func New(t testing.TB) *Client {
	return &Client{&mock.Expector{T: t}}
}

// RegisterDeviceForPush implements notification.Client
func (c *Client) RegisterDeviceForPush(in *notification.DeviceRegistrationInfo) error {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

// DeregisterDeviceForPush implements notification.Client
func (c *Client) DeregisterDeviceForPush(deviceID string) error {
	rets := c.Expector.Record(deviceID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

// SendNotification implements notification.Client
func (c *Client) SendNotification(in *notification.Notification) error {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}
