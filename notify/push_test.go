package notify

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/test"
)

func init() {
	dispatch.Testing = true
	conc.Testing = true
}

type pushMockDataAPI struct {
	api.DataAPI

	pushConfigData []*common.PushConfigData

	deletePushCalled bool
}

type pushMockAuthAPI struct {
	api.AuthAPI
}

func (m *pushMockDataAPI) GetPushConfigDataForAccount(accountID int64) ([]*common.PushConfigData, error) {
	return m.pushConfigData, nil
}
func (m *pushMockDataAPI) DeletePushCommunicationPreferenceForAccount(accountID int64) error {
	m.deletePushCalled = true
	return nil
}

func TestPushNotificationsLive(t *testing.T) {
	endpoint := os.Getenv("TEST_PUSH_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_PUSH_ENDPOINT not set")
	}

	awsConfig := &aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewEnvCredentials(),
	}
	snsCli := sns.New(session.New(awsConfig))

	dataAPI := &pushMockDataAPI{
		pushConfigData: []*common.PushConfigData{{
			PushEndpoint:   endpoint,
			Platform:       device.IOS,
			AppType:        "patient",
			AppEnvironment: "staging",
		}},
	}
	authAPI := &pushMockAuthAPI{}
	configs := &config.NotificationConfigs{
		"iOS-patient-staging": &config.NotificationConfig{
			Platform: device.IOS,
		},
	}
	m := NewManager(dataAPI, authAPI, snsCli, nil, nil, "", configs, metrics.NewRegistry())
	if err := m.pushNotificationToUser(1, api.RolePatient, &Message{ShortMessage: "test"}, 3); err != nil {
		t.Fatal(err)
	}
}

type awsErr struct{}

func (a *awsErr) Code() string {
	return "EndpointDisabled"
}
func (a *awsErr) Error() string {
	return "error"
}

type mockSNSAPI_endpointdisabled struct {
	snsiface.SNSAPI
}

func (m *mockSNSAPI_endpointdisabled) Publish(*sns.PublishInput) (*sns.PublishOutput, error) {
	return nil, &awsErr{}
}

func TestPushNotifications_EndpointDisabled(t *testing.T) {
	conc.Testing = true
	dataAPI := &pushMockDataAPI{
		pushConfigData: []*common.PushConfigData{{
			Platform:       device.IOS,
			AppType:        "patient",
			AppEnvironment: "staging",
		}},
	}
	authAPI := &pushMockAuthAPI{}
	configs := &config.NotificationConfigs{
		"iOS-patient-staging": &config.NotificationConfig{
			Platform: device.IOS,
		},
	}
	m := NewManager(dataAPI, authAPI, &mockSNSAPI_endpointdisabled{}, nil, nil, "", configs, metrics.NewRegistry())
	if err := m.pushNotificationToUser(1, api.RolePatient, &Message{ShortMessage: "test"}, 3); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, true, dataAPI.deletePushCalled)
}
