package server

import (
	"context"
	"testing"

	dalmock "github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/hint"
	hintoauthmock "github.com/sprucehealth/backend/libs/hintutils/mock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/patientsync"
)

func TestConfigureSync(t *testing.T) {
	dmock := dalmock.New(t)
	defer dmock.Finish()

	omock := hintoauthmock.NewOAuthClient(t)
	defer omock.Finish()

	hint.SetOAuthClient(omock)

	s := &server{
		dl: dmock,
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

	omock.Expect(mock.NewExpectation(omock.GrantAPIKey, "token123").WithReturns(&hint.PracticeGrant{
		AccessToken: "accessToken123",
		Practice: &hint.Practice{
			ID: "prac-1",
		},
	}, nil))

	_, err := s.ConfigureSync(context.Background(), &patientsync.ConfigureSyncRequest{
		OrganizationEntityID: "orgID",
		Source:               patientsync.SOURCE_HINT,
		ThreadType:           patientsync.THREAD_TYPE_SECURE,
		Token:                "token123",
	})
	test.OK(t, err)

}
