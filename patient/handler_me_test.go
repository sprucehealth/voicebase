package patient

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
)

func init() {
	apiservice.Testing = true
}

type mockAPIMeHandler struct {
	api.DataAPI
	feedbackRecorded bool
	tp               []*common.TreatmentPlan
}

func (m *mockAPIMeHandler) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
	return &common.Patient{AccountID: encoding.DeprecatedNewObjectID(1), ID: common.NewPatientID(1)}, nil
}

func (m *mockAPIMeHandler) PatientFeedbackRecorded(patientID common.PatientID, feedbackFor string) (bool, error) {
	return m.feedbackRecorded, nil
}

func (m *mockAPIMeHandler) GetActiveTreatmentPlansForPatient(patientID common.PatientID) ([]*common.TreatmentPlan, error) {
	return m.tp, nil
}

func TestMeHandlerFeedback(t *testing.T) {
	mockAPI := &mockAPIMeHandler{}
	handler := NewMeHandler(mockAPI, dispatch.New())

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

	// Shouldn't show feedback prompt is already recorded

	mockAPI.feedbackRecorded = true

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
}
