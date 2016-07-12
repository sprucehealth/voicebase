package patient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/feedback"
	"github.com/sprucehealth/backend/cmd/svc/restapi/tagging"
	"github.com/sprucehealth/backend/cmd/svc/restapi/tagging/model"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
)

type mockDataAPI_feedback struct {
	api.DataAPI
	tps []*common.TreatmentPlan
}

func (m *mockDataAPI_feedback) GetPatientIDFromAccountID(accountID int64) (common.PatientID, error) {
	return common.PatientID{
		ObjectID: encoding.NewObjectID(1),
	}, nil
}
func (m *mockDataAPI_feedback) GetActiveTreatmentPlansForPatient(patientID common.PatientID) ([]*common.TreatmentPlan, error) {
	return m.tps, nil
}

type mockFeedbackClient_feedback struct {
	feedback.DAL

	f                          *feedback.FeedbackTemplateData
	feedbackRecorded           bool
	structuredResponseRecorded feedback.StructuredResponse
}

func (m *mockFeedbackClient_feedback) FeedbackTemplate(id int64) (*feedback.FeedbackTemplateData, error) {
	return m.f, nil
}
func (m *mockFeedbackClient_feedback) RecordPatientFeedback(patientID common.PatientID, feedbackFor string, rating int, comment *string, res feedback.StructuredResponse) error {
	m.feedbackRecorded = true
	m.structuredResponseRecorded = res
	return nil
}

type mockTaggingClient_feedback struct {
	tagging.Client
}

func (t *mockTaggingClient_feedback) InsertTagAssociation(tag *model.Tag, membership *model.TagMembership) (int64, error) {
	return 0, nil
}

func TestFeedbackIntake(t *testing.T) {
	conc.Testing = true
	md := &mockDataAPI_feedback{
		tps: []*common.TreatmentPlan{
			{
				SentDate: ptr.Time(time.Now()),
			},
		},
	}

	mf := &mockFeedbackClient_feedback{
		f: &feedback.FeedbackTemplateData{
			ID:       10,
			Template: &feedback.FreeTextTemplate{},
		},
	}

	cfgStore, err := cfg.NewLocalStore(nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewFeedbackHandler(md, mf, &mockTaggingClient_feedback{}, cfgStore)

	jsonData, err := json.Marshal(&feedback.FreeTextResponse{
		Response: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}

	requestBody := feedbackSubmitRequest{
		Rating: 5,
		AdditionalFeedback: &additionalFeedback{
			TemplateID: 10,
			Answer:     json.RawMessage(jsonData),
		},
	}

	jsonData, err = json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("POST", "/", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	ctx := context.Background()
	ctx = apiservice.CtxWithAccount(ctx, &common.Account{ID: 1, Role: api.RolePatient})

	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// ensure that feedback was recorded
	test.Equals(t, true, mf.feedbackRecorded)
	_, ok := mf.structuredResponseRecorded.(*feedback.FreeTextResponse)
	test.Equals(t, true, ok)
}
