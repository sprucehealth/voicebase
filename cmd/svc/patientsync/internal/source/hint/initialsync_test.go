package hint

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	patientmock "github.com/sprucehealth/backend/libs/hintutils/mock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/go-hint"
)

func TestInitialSync(t *testing.T) {
	pmock := patientmock.New(t)
	defer pmock.Finish()

	hint.SetPatientClient(pmock)

	dmock := dalmock.New(t)
	defer dmock.Finish()

	mocksqsAPI := mock.NewSQSAPI(t)
	defer mocksqsAPI.Finish()

	dmock.Expect(mock.NewExpectation(dmock.SyncConfigForOrg, "orgID", "SOURCE_HINT").WithReturns(&sync.Config{
		OrganizationEntityID: "orgID",
		Source:               sync.SOURCE_HINT,
		Token: &sync.Config_Hint{
			Hint: &sync.HintToken{
				AccessToken: "accessToken",
			},
		},
	}, nil))

	bookmarkTime := time.Now().Add(-2 * time.Hour)
	lastPatientCreatedAt := time.Now().Add(-1 * time.Hour)
	dmock.Expect(mock.NewExpectation(dmock.SyncBookmarkForOrg, "orgID").WithReturns(&dal.SyncBookmark{
		Bookmark: bookmarkTime,
		Status:   dal.SyncStatusInitiated,
	}, nil))

	params := &hint.ListParams{
		Sort: &hint.Sort{
			By: "created_at",
		},
		Items: []*hint.QueryItem{
			{
				Field: "created_at",
				Operations: []*hint.Operation{
					{
						Operator: hint.OperatorGreaterThan,
						Operand:  bookmarkTime.String(),
					},
				},
			},
		},
	}

	patients := []interface{}{
		&hint.Patient{
			ID:        "pat-test",
			FirstName: "FirstName1",
			LastName:  "LastName1",
			Email:     "firstname1@example.com",
			CreatedAt: lastPatientCreatedAt,
			Phones: []*hint.Phone{
				{
					Type:   hint.PhoneTypeMobile,
					Number: "+12068773590",
				},
			},
		},
		&hint.Patient{
			ID:        "pat-test2",
			FirstName: "FirstName2",
			LastName:  "LastName2",
			Email:     "firstname2@example.com",
			CreatedAt: lastPatientCreatedAt,
			Phones: []*hint.Phone{
				{
					Type:   hint.PhoneTypeMobile,
					Number: "+13068773590",
				},
			},
		},
	}
	pmock.Expect(mock.NewExpectation(pmock.List, "accessToken", params).WithReturns(hint.GetIter(params, func(params *hint.ListParams) ([]interface{}, hint.ListMeta, error) {
		defer func() {
			patients = nil
		}()
		return patients, hint.ListMeta{}, nil
	})))

	syncPatients := &sync.Event{
		Source:               sync.SOURCE_HINT,
		Type:                 sync.EVENT_TYPE_PATIENT_ADD,
		OrganizationEntityID: "orgID",
		Event: &sync.Event_PatientAddEvent{
			PatientAddEvent: &sync.PatientAddEvent{
				Patients: []*sync.Patient{
					{
						ID:             "pat-test",
						FirstName:      "FirstName1",
						LastName:       "LastName1",
						EmailAddresses: []string{"firstname1@example.com"},
						ExternalURL:    "https://provider.hint.com/patients/pat-test",

						PhoneNumbers: []*sync.Phone{
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+12068773590",
							},
						},
					},
					{
						ID:             "pat-test2",
						FirstName:      "FirstName2",
						LastName:       "LastName2",
						EmailAddresses: []string{"firstname2@example.com"},
						ExternalURL:    "https://provider.hint.com/patients/pat-test2",

						PhoneNumbers: []*sync.Phone{
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+13068773590",
							},
						},
					},
				},
			},
		},
	}
	data, err := syncPatients.Marshal()
	test.OK(t, err)
	msg := base64.StdEncoding.EncodeToString(data)

	mocksqsAPI.Expect(mock.NewExpectation(mocksqsAPI.SendMessage, &sqs.SendMessageInput{
		MessageBody: &msg,
		QueueUrl:    ptr.String(""),
	}))

	dmock.Expect(mock.NewExpectation(dmock.UpdateSyncBookmarkForOrg, "orgID", lastPatientCreatedAt, dal.SyncStatusConnected))

	test.OK(t, DoInitialSync(dmock, "orgID", "", mocksqsAPI))

}
