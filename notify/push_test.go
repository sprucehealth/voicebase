package notify

import (
	"os"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/dispatch"
)

func init() {
	dispatch.Testing = true
}

type pushMockAPI struct {
	api.DataAPI
	api.AuthAPI

	pushConfigData []*common.PushConfigData
}

func (m *pushMockAPI) GetPushConfigDataForAccount(accountID int64) ([]*common.PushConfigData, error) {
	return m.pushConfigData, nil
}

func TestPushNotificationsLive(t *testing.T) {
	endpoint := os.Getenv("TEST_PUSH_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_PUSH_ENDPOINT not set")
	}

	awsConfig := &aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewEnvCredentials(),
	}
	snsCli := sns.New(awsConfig)

	mockAPI := &pushMockAPI{
		pushConfigData: []*common.PushConfigData{{
			PushEndpoint:   endpoint,
			Platform:       common.IOS,
			AppType:        "patient",
			AppEnvironment: "staging",
		}},
	}
	configs := &config.NotificationConfigs{
		"iOS-patient-staging": &config.NotificationConfig{
			Platform: common.IOS,
		},
	}
	m := NewManager(mockAPI, mockAPI, snsCli, nil, nil, "", configs, metrics.NewRegistry())
	if err := m.pushNotificationToUser(1, api.RolePatient, &Message{ShortMessage: "test"}, 3); err != nil {
		t.Fatal(err)
	}
}
