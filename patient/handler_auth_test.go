package patient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ratelimit"
)

type mockAPIAuthenticationHandler struct {
	api.DataAPI
	api.AuthAPI
	feedbackRecorded bool
	tp               []*common.TreatmentPlan
}

func (m *mockAPIAuthenticationHandler) Authenticate(login, password string) (*common.Account, error) {
	return &common.Account{ID: 1}, nil
}

func (m *mockAPIAuthenticationHandler) CreateToken(accountID int64, platform api.Platform, opt api.CreateTokenOption) (string, error) {
	return "TOKEN", nil
}

func (m *mockAPIAuthenticationHandler) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
	return &common.Patient{AccountID: encoding.NewObjectID(1), ID: encoding.NewObjectID(1)}, nil
}

func (m *mockAPIAuthenticationHandler) PatientFeedbackRecorded(patientID int64, feedbackFor string) (bool, error) {
	return m.feedbackRecorded, nil
}

func (m *mockAPIAuthenticationHandler) GetActiveTreatmentPlansForPatient(patientID int64) ([]*common.TreatmentPlan, error) {
	return m.tp, nil
}

func TestAuthenticationHandlerFeedback(t *testing.T) {
	mockAPI := &mockAPIAuthenticationHandler{}
	handler := NewAuthenticationHandler(mockAPI, mockAPI, dispatch.New(), "", ratelimit.NullKeyed{}, metrics.NewRegistry())

	// No treatment plans so shouldn't show feedback

	var res AuthenticationResponse
	err := testJSONHandler(handler,
		newJSONTestRequest("POST", "/x/authenticate", &AuthRequestData{Login: "X", Password: "Y"}),
		&res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ActionsNeeded) != 0 {
		t.Fatalf("Expected no actions needed, got %d", len(res.ActionsNeeded))
	}

	// Unviewed treatment plan shouldn't trigger feedback

	tm := time.Now()
	mockAPI.tp = []*common.TreatmentPlan{{ID: encoding.NewObjectID(1), PatientCaseID: encoding.NewObjectID(1), PatientViewed: false, SentDate: &tm}}

	res = AuthenticationResponse{}
	err = testJSONHandler(handler,
		newJSONTestRequest("POST", "/x/authenticate", &AuthRequestData{Login: "X", Password: "Y"}),
		&res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ActionsNeeded) != 0 {
		t.Fatalf("Expected no actions needed, got %d", len(res.ActionsNeeded))
	}

	// Viewed treatment plan should show feedback since hasn't been recorded yet

	mockAPI.tp = []*common.TreatmentPlan{{ID: encoding.NewObjectID(1), PatientCaseID: encoding.NewObjectID(1), PatientViewed: true, SentDate: &tm}}

	res = AuthenticationResponse{}
	err = testJSONHandler(handler,
		newJSONTestRequest("POST", "/x/authenticate", &AuthRequestData{Login: "X", Password: "Y"}),
		&res)
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

	res = AuthenticationResponse{}
	err = testJSONHandler(handler,
		newJSONTestRequest("POST", "/x/authenticate", &AuthRequestData{Login: "X", Password: "Y"}),
		&res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ActionsNeeded) != 0 {
		t.Fatalf("Expected no actions needed, got %d", len(res.ActionsNeeded))
	}
}

func newJSONTestRequest(method, path string, body interface{}) *http.Request {
	var bodyReader io.Reader
	if body != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			panic(err)
		}
		bodyReader = buf
	}
	rq, err := http.NewRequest(method, path, bodyReader)
	if err != nil {
		panic(err)
	}
	if bodyReader != nil {
		rq.Header.Set("Content-Type", httputil.JSONContentType)
	}
	return rq
}

func testJSONHandler(handler http.Handler, req *http.Request, res interface{}) error {
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		return fmt.Errorf("Expected status %d, got %d", http.StatusOK, rw.Code)
	}
	return json.NewDecoder(rw.Body).Decode(res)
}
