package hint

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/go-hint"
)

func TestWebhookHandler(t *testing.T) {

	dmock := dalmock.New(t)
	defer dmock.Finish()

	mocksqsAPI := mock.NewSQSAPI(t)
	defer mocksqsAPI.Finish()

	patient := &hint.Patient{
		ID:        "pat-test",
		FirstName: "Joe",
		LastName:  "Schmoe",
		Email:     "joe@schmoe.com",
		Phones: []*hint.Phone{
			{
				Type:   hint.PhoneTypeHome,
				Number: "+17348465522",
			},
			{
				Type:   hint.PhoneTypeMobile,
				Number: "+12068773590",
			},
		},
	}

	jsonData, err := json.Marshal(patient)
	test.OK(t, err)

	ev := &event{
		ID:         "evt-jKi2jlalOJk3",
		CreatedAt:  time.Now(),
		Type:       "patient.created",
		PracticeID: "prac-123",
		Object:     json.RawMessage(jsonData),
	}

	data, err := json.Marshal(ev)
	test.OK(t, err)

	r, err := http.NewRequest("POST", "test", bytes.NewReader(data))
	test.OK(t, err)

	dmock.Expect(mock.NewExpectation(dmock.SyncConfigForExternalID, "prac-123").WithReturns(&sync.Config{
		Source:               sync.SOURCE_HINT,
		OrganizationEntityID: "orgID",
	}, nil))

	dmock.Expect(mock.NewExpectation(dmock.SyncBookmarkForOrg, "orgID").WithReturns(&dal.SyncBookmark{
		Status: dal.SyncStatusConnected,
	}, nil))

	syncPatients := &sync.Event{
		Source:               sync.SOURCE_HINT,
		OrganizationEntityID: "orgID",
		Event: &sync.Event_PatientAddEvent{
			PatientAddEvent: &sync.PatientAddEvent{
				Patients: []*sync.Patient{
					{
						ID:             "pat-test",
						FirstName:      "Joe",
						LastName:       "Schmoe",
						EmailAddresses: []string{"joe@schmoe.com"},
						ExternalURL:    "https://provider.hint.com/patients/pat-test",
						PhoneNumbers: []*sync.Phone{
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+12068773590",
							},
							{
								Type:   sync.PHONE_TYPE_HOME,
								Number: "+17348465522",
							},
						},
						CreatedTime:      uint64(patient.CreatedAt.Unix()),
						LastModifiedTime: uint64(patient.UpdatedAt.Unix()),
					},
				},
			},
		},
	}
	syncData, err := syncPatients.Marshal()
	test.OK(t, err)
	msg := base64.StdEncoding.EncodeToString(syncData)

	mocksqsAPI.Expect(mock.NewExpectation(mocksqsAPI.SendMessage, &sqs.SendMessageInput{
		MessageBody: &msg,
		QueueUrl:    ptr.String("queueURL"),
	}))

	w := &webhookHandler{
		dl:                 dmock,
		syncEventsQueueURL: "queueURL",
		sqsAPI:             mocksqsAPI,
	}

	recorder := httptest.NewRecorder()

	w.ServeHTTP(recorder, r)
	test.Equals(t, http.StatusOK, recorder.Code)

}
