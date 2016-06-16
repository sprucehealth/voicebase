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

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/feedback"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"golang.org/x/net/context"
)

type mockDataAPIAuthenticationHandler struct {
	api.DataAPI
	tp []*common.TreatmentPlan
}

type mockFeedbackClient struct {
	feedback.DAL
	feedbackRecorded     bool
	pendingRecordCreated bool
	askForFeedback       bool
}

func (m *mockFeedbackClient) PatientFeedbackRecorded(patientID common.PatientID, feedbackFor string) (bool, error) {
	return m.feedbackRecorded, nil
}
func (m *mockFeedbackClient) CreatePendingPatientFeedback(patientID common.PatientID, feedbackFor string) error {
	m.pendingRecordCreated = true
	return nil
}
func (m *mockFeedbackClient) Show(tp *common.TreatmentPlan) bool {
	return m.askForFeedback
}

func (m *mockDataAPIAuthenticationHandler) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
	return &common.Patient{AccountID: encoding.DeprecatedNewObjectID(1), ID: common.NewPatientID(1)}, nil
}

func (m *mockDataAPIAuthenticationHandler) GetActiveTreatmentPlansForPatient(patientID common.PatientID) ([]*common.TreatmentPlan, error) {
	return m.tp, nil
}

type mockAuthAPIAuthenticationHandler struct {
	api.AuthAPI
}

func (m *mockAuthAPIAuthenticationHandler) Authenticate(login, password string) (*common.Account, error) {
	return &common.Account{ID: 1}, nil
}

func (m *mockAuthAPIAuthenticationHandler) CreateToken(accountID int64, platform api.Platform, opt api.CreateTokenOption) (string, error) {
	return "TOKEN", nil
}

func TestAuthenticationHandlerFeedback(t *testing.T) {
	conc.Testing = true
	dataAPI := &mockDataAPIAuthenticationHandler{}
	authAPI := &mockAuthAPIAuthenticationHandler{}
	fClient := &mockFeedbackClient{}
	handler := NewAuthenticationHandler(dataAPI, authAPI, fClient, dispatch.New(), "", ratelimit.NullKeyed{}, metrics.NewRegistry())

	// No treatment plans so shouldn't show feedback

	var res AuthenticationResponse
	err := testJSONHandler(handler,
		context.Background(),
		newJSONTestRequest("POST", "/x/authenticate", &AuthRequestData{Login: "X", Password: "Y"}),
		&res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ActionsNeeded) != 0 {
		t.Fatalf("Expected no actions needed, got %d", len(res.ActionsNeeded))
	}
	if fClient.pendingRecordCreated {
		t.Fatal("Expected no record to be created")
	}

	// Unviewed treatment plan shouldn't trigger feedback

	tm := time.Now()
	dataAPI.tp = []*common.TreatmentPlan{{ID: encoding.DeprecatedNewObjectID(1), PatientCaseID: encoding.DeprecatedNewObjectID(1), PatientViewed: false, SentDate: &tm}}

	res = AuthenticationResponse{}
	err = testJSONHandler(handler,
		context.Background(),
		newJSONTestRequest("POST", "/x/authenticate", &AuthRequestData{Login: "X", Password: "Y"}),
		&res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ActionsNeeded) != 0 {
		t.Fatalf("Expected no actions needed, got %d", len(res.ActionsNeeded))
	}
	if fClient.pendingRecordCreated {
		t.Fatal("Expected no pending feedback record to be created")
	}

	// Viewed treatment plan should show feedback since hasn't been recorded yet

	dataAPI.tp = []*common.TreatmentPlan{{ID: encoding.DeprecatedNewObjectID(1), PatientCaseID: encoding.DeprecatedNewObjectID(1), PatientViewed: true, SentDate: &tm}}

	res = AuthenticationResponse{}
	err = testJSONHandler(handler,
		context.Background(),
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
	if !fClient.pendingRecordCreated {
		t.Fatal("Expected pending record to be created but got none")
	}

	// Shouldn't show feedback prompt is already recorded

	fClient.feedbackRecorded = true
	fClient.pendingRecordCreated = false

	res = AuthenticationResponse{}
	err = testJSONHandler(handler,
		context.Background(),
		newJSONTestRequest("POST", "/x/authenticate", &AuthRequestData{Login: "X", Password: "Y"}),
		&res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ActionsNeeded) != 0 {
		t.Fatalf("Expected no actions needed, got %d", len(res.ActionsNeeded))
	}
	if fClient.pendingRecordCreated {
		t.Fatal("Expected no pending record to be created")
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

func testJSONHandler(handler httputil.ContextHandler, ctx context.Context, req *http.Request, res interface{}) error {
	rw := httptest.NewRecorder()
	handler.ServeHTTP(ctx, rw, req)
	if rw.Code != http.StatusOK {
		return fmt.Errorf("Expected status %d, got %d", http.StatusOK, rw.Code)
	}
	return json.NewDecoder(rw.Body).Decode(res)
}
