package server

import (
	"context"
	"testing"

	dalmock "github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	psettings "github.com/sprucehealth/backend/cmd/svc/patientsync/settings"
	hintoauthmock "github.com/sprucehealth/backend/libs/hintutils/mock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/settings"
	settingsmock "github.com/sprucehealth/backend/svc/settings/mock"
	"github.com/sprucehealth/go-hint"
)

func TestConfigureSync(t *testing.T) {
	dmock := dalmock.New(t)
	defer dmock.Finish()

	omock := hintoauthmock.NewOAuthClient(t)
	defer omock.Finish()

	smock := settingsmock.New(t)
	defer smock.Finish()

	hint.SetOAuthClient(omock)

	s := &server{
		dl:       dmock,
		settings: smock,
	}

	dmock.Expect(mock.NewExpectation(dmock.CreateSyncConfig, &sync.Config{
		OrganizationEntityID: "orgID",
		Source:               sync.SOURCE_HINT,
		ThreadCreationType:   sync.THREAD_CREATION_TYPE_SECURE,
		Token: &sync.Config_Hint{
			Hint: &sync.HintToken{
				AccessToken:  "accessToken123",
				RefreshToken: "",
				ExpiresIn:    0,
				PracticeID:   "prac-1",
			},
		},
	}, ptr.String("prac-1")))

	smock.Expect(mock.NewExpectation(smock.GetValues, &settings.GetValuesRequest{
		NodeID: "orgID",
		Keys: []*settings.ConfigKey{
			{
				Key: psettings.ConfigKeyThreadTypeOption,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Value: &settings.Value_SingleSelect{
					SingleSelect: &settings.SingleSelectValue{
						Item: &settings.ItemValue{
							ID: psettings.ThreadTypeOptionSecure,
						},
					},
				},
			},
		},
	}, nil))

	omock.Expect(mock.NewExpectation(omock.GrantAPIKey, "token123").WithReturns(&hint.PracticeGrant{
		AccessToken: "accessToken123",
		Practice: &hint.Practice{
			ID: "prac-1",
		},
	}, nil))

	_, err := s.ConfigureSync(context.Background(), &patientsync.ConfigureSyncRequest{
		OrganizationEntityID: "orgID",
		Source:               patientsync.SOURCE_HINT,
		Token:                "token123",
	})
	test.OK(t, err)

}
