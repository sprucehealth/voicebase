package patient

import (
	"testing"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/feedback"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dispatch"
)

func init() {
	apiservice.Testing = true
}

type mockAPIMeHandler struct {
	api.DataAPI

	tp []*common.TreatmentPlan
}

type mockFeedbackClient_Me struct {
	feedback.DAL
	feedbackRecorded     bool
	pendingRecordCreated bool
}

func (m *mockFeedbackClient_Me) PatientFeedbackRecorded(patientID common.PatientID, feedbackFor string) (bool, error) {
	return m.feedbackRecorded, nil
}

func (m *mockFeedbackClient_Me) CreatePendingPatientFeedback(patientID common.PatientID, feedbackFor string) error {
	m.pendingRecordCreated = true
	return nil
}

func (m *mockAPIMeHandler) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
	return &common.Patient{AccountID: encoding.DeprecatedNewObjectID(1), ID: common.NewPatientID(1)}, nil
}

func (m *mockAPIMeHandler) GetActiveTreatmentPlansForPatient(patientID common.PatientID) ([]*common.TreatmentPlan, error) {
	return m.tp, nil
}

func TestMeHandlerFeedback(t *testing.T) {
	conc.Testing = true
	mockAPI := &mockAPIMeHandler{}
	fClient := &mockFeedbackClient_Me{}
	handler := NewMeHandler(mockAPI, fClient, dispatch.New())
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RolePatient, ID: 1})

	// No treatment plans so shouldn't show feedback

	var res meResponse
	req := newJSONTestRequest("GET", "/", nil)
	req.Header.Set("Authorization", "token abc")
	err := testJSONHandler(handler, ctx, req, &res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ActionsNeeded) != 0 {
		t.Fatalf("Expected no actions needed, got %d", len(res.ActionsNeeded))
	}

	// Unviewed treatment plan shouldn't trigger feedback

	tm := time.Now()
	mockAPI.tp = []*common.TreatmentPlan{{ID: encoding.DeprecatedNewObjectID(1), PatientCaseID: encoding.DeprecatedNewObjectID(1), PatientViewed: false, SentDate: &tm}}

	res = meResponse{}
	req = newJSONTestRequest("GET", "/", nil)
	req.Header.Set("Authorization", "token abc")
	err = testJSONHandler(handler, ctx, req, &res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ActionsNeeded) != 0 {
		t.Fatalf("Expected no actions needed, got %d", len(res.ActionsNeeded))
	}

	// Viewed treatment plan should show feedback since hasn't been recorded yet

	mockAPI.tp = []*common.TreatmentPlan{{ID: encoding.DeprecatedNewObjectID(1), PatientCaseID: encoding.DeprecatedNewObjectID(1), PatientViewed: true, SentDate: &tm}}

	res = meResponse{}
	req = newJSONTestRequest("GET", "/", nil)
	req.Header.Set("Authorization", "token abc")
	err = testJSONHandler(handler, ctx, req, &res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ActionsNeeded) != 1 {
		t.Fatalf("Expected 1 action needed, got %d", len(res.ActionsNeeded))
	}
	if res.ActionsNeeded[0].Type != actionNeededSimpleFeedbackPrompt {
		t.Fatalf("Expected action needed of '%s', got '%s'", actionNeededSimpleFeedbackPrompt, res.ActionsNeeded[0].Type)
	}
	if !fClient.pendingRecordCreated {
		t.Fatal("Expected a pending record for feedback to be created but none was")
	}

	// Shouldn't show feedback prompt is already recorded

	fClient.feedbackRecorded = true
	fClient.pendingRecordCreated = false

	res = meResponse{}
	req = newJSONTestRequest("GET", "/", nil)
	req.Header.Set("Authorization", "token abc")
	err = testJSONHandler(handler, ctx, req, &res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ActionsNeeded) != 0 {
		t.Fatalf("Expected no actions needed, got %d", len(res.ActionsNeeded))
	}
	if fClient.pendingRecordCreated {
		t.Fatal("Expected no pending record to be created but one weas q")
	}
}
