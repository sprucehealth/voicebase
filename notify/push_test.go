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
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dispatch"
)

func init() {
	dispatch.Testing = true
	conc.Testing = true
}

type pushMockDataAPI struct {
	api.DataAPI

	pushConfigData []*common.PushConfigData
}

type pushMockAuthAPI struct {
	api.AuthAPI
}

func (m *pushMockDataAPI) GetPushConfigDataForAccount(accountID int64) ([]*common.PushConfigData, error) {
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

	dataAPI := &pushMockDataAPI{
		pushConfigData: []*common.PushConfigData{{
			PushEndpoint:   endpoint,
			Platform:       common.IOS,
			AppType:        "patient",
			AppEnvironment: "staging",
		}},
	}
	authAPI := &pushMockAuthAPI{}
	configs := &config.NotificationConfigs{
		"iOS-patient-staging": &config.NotificationConfig{
			Platform: common.IOS,
		},
	}
	m := NewManager(dataAPI, authAPI, snsCli, nil, nil, "", configs, metrics.NewRegistry())
	if err := m.pushNotificationToUser(1, api.RolePatient, &Message{ShortMessage: "test"}, 3); err != nil {
		t.Fatal(err)
	}
}
