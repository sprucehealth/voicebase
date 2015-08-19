package appevent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type mockDataAPIHandler struct {
	api.DataAPI
}

func (a *mockDataAPIHandler) GetPatientFromTreatmentPlanID(tpID int64) (*common.Patient, error) {
	return &common.Patient{
		AccountID: encoding.NewObjectID(1),
	}, nil
}

func (a *mockDataAPIHandler) GetCaseIDFromMessageID(msgID int64) (int64, error) {
	return msgID, nil
}

func (a *mockDataAPIHandler) GetPatientCaseFromID(caseID int64) (*common.PatientCase, error) {
	return &common.PatientCase{
		PatientID: common.NewPatientID(uint64(caseID)),
	}, nil
}

func (a *mockDataAPIHandler) Patient(id common.PatientID, basic bool) (*common.Patient, error) {
	return &common.Patient{AccountID: encoding.NewObjectID(id.Uint64())}, nil
}

func (a *mockDataAPIHandler) GetDoctorIDFromAccountID(accountID int64) (int64, error) {
	return accountID, nil
}

func (a *mockDataAPIHandler) GetDoctorsAssignedToPatientCase(patientCaseID int64) ([]*common.CareProviderAssignment, error) {
	return []*common.CareProviderAssignment{
		{ProviderRole: api.RoleDoctor, ProviderID: 1, Status: api.StatusTemp, Expires: ptr.Time(time.Now().Add(time.Hour))},
	}, nil
}

func TestHandler(t *testing.T) {
	dataAPI := &mockDataAPIHandler{}
	dispatcher := dispatch.New()
	h := NewHandler(dataAPI, dispatcher)

	account := &common.Account{ID: 1}
	ctx := context.Background()
	ctx = apiservice.CtxWithAccount(ctx, account)

	cases := []struct {
		request        EventRequestData
		account        common.Account
		expectedStatus int
	}{
		{
			request:        EventRequestData{Action: ViewedAction, Resource: "treatment_plan", ResourceID: 1},
			account:        common.Account{ID: 123, Role: api.RolePatient},
			expectedStatus: http.StatusForbidden,
		},
		{
			request:        EventRequestData{Action: ViewedAction, Resource: "treatment_plan", ResourceID: 1},
			account:        common.Account{ID: 1, Role: api.RolePatient},
			expectedStatus: http.StatusOK,
		},
		{
			request:        EventRequestData{Action: ViewedAction, Resource: "case_message", ResourceID: 1},
			account:        common.Account{ID: 123, Role: api.RolePatient},
			expectedStatus: http.StatusForbidden,
		},
		{
			request:        EventRequestData{Action: ViewedAction, Resource: "case_message", ResourceID: 1},
			account:        common.Account{ID: 1, Role: api.RolePatient},
			expectedStatus: http.StatusOK,
		},
		{
			request:        EventRequestData{Action: ViewedAction, Resource: "case_message", ResourceID: 1},
			account:        common.Account{ID: 123, Role: api.RoleDoctor},
			expectedStatus: http.StatusForbidden,
		},
		{
			request:        EventRequestData{Action: ViewedAction, Resource: "case_message", ResourceID: 1},
			account:        common.Account{ID: 1, Role: api.RoleDoctor},
			expectedStatus: http.StatusOK,
		},
		{
			request:        EventRequestData{Action: ViewedAction, Resource: "all_case_messages", ResourceID: 1},
			account:        common.Account{ID: 123, Role: api.RolePatient},
			expectedStatus: http.StatusForbidden,
		},
		{
			request:        EventRequestData{Action: ViewedAction, Resource: "all_case_messages", ResourceID: 1},
			account:        common.Account{ID: 1, Role: api.RolePatient},
			expectedStatus: http.StatusOK,
		},
		{
			request:        EventRequestData{Action: ViewedAction, Resource: "all_case_messages", ResourceID: 1},
			account:        common.Account{ID: 123, Role: api.RoleDoctor},
			expectedStatus: http.StatusForbidden,
		},
		{
			request:        EventRequestData{Action: ViewedAction, Resource: "all_case_messages", ResourceID: 1},
			account:        common.Account{ID: 1, Role: api.RoleDoctor},
			expectedStatus: http.StatusOK,
		},
	}

	for _, c := range cases {
		t.Logf("Test case: %+v", c)
		body, err := json.Marshal(c.request)
		test.OK(t, err)
		*account = c.account
		r, err := http.NewRequest(httputil.Post, "/", bytes.NewReader(body))
		test.OK(t, err)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(ctx, w, r)
		test.HTTPResponseCode(t, c.expectedStatus, w)
	}
}
