package notification

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/test"
)

var clientConfig = &ClientConfig{
	SQSDeviceDeregistrationURL: "SQSDeviceDeregistrationURL",
	SQSDeviceRegistrationURL:   "SQSDeviceRegistrationURL",
	SQSNotificationURL:         "SQSNotificationURL",
}

func TestDeviceRegistration(t *testing.T) {
	sqsAPI := mock.NewSQSAPI(t)
	defer sqsAPI.Finish()
	client := NewClient(sqsAPI, clientConfig)
	dri := &DeviceRegistrationInfo{DeviceToken: "token"}
	expectSendMessage(t, sqsAPI, dri, clientConfig.SQSDeviceRegistrationURL)
	client.RegisterDeviceForPush(dri)
}

func TestDeviceDeregistration(t *testing.T) {
	sqsAPI := mock.NewSQSAPI(t)
	defer sqsAPI.Finish()
	client := NewClient(sqsAPI, clientConfig)
	ddri := &DeviceDeregistrationInfo{DeviceID: "deviceID"}
	expectSendMessage(t, sqsAPI, ddri, clientConfig.SQSDeviceDeregistrationURL)
	client.DeregisterDeviceForPush("deviceID")
}

func TestSendNotification(t *testing.T) {
	sqsAPI := mock.NewSQSAPI(t)
	defer sqsAPI.Finish()
	client := NewClient(sqsAPI, clientConfig)
	n := &Notification{CollapseKey: "collapse"}
	expectSendMessage(t, sqsAPI, n, clientConfig.SQSNotificationURL)
	client.SendNotification(n)
}

func expectSendMessage(t *testing.T, sqsAPI *mock.SQSAPI, message interface{}, queueURL string) {
	body, err := json.Marshal(message)
	test.OK(t, err)
	sqsAPI.Expect(mock.NewExpectation(sqsAPI.SendMessage, &sqs.SendMessageInput{
		MessageBody: ptr.String(string(body)),
		QueueUrl:    ptr.String(queueURL),
	}))
}
