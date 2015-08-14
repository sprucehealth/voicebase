package patient_visit

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type mockConsentDataAPI struct {
	api.DataAPI
	visit *common.PatientVisit
}

func (m *mockConsentDataAPI) GetPatientIDFromAccountID(accountID int64) (common.PatientID, error) {
	return common.NewPatientID(1), nil
}
func (m *mockConsentDataAPI) GetPatientVisitFromID(visitID int64) (*common.PatientVisit, error) {
	return m.visit, nil
}
func (m *mockConsentDataAPI) UpdatePatientVisit(visitID int64, update *api.PatientVisitUpdate) (int, error) {
	if *update.RequiredStatus != m.visit.Status {
		return 0, errors.New("visit status does not match " + m.visit.Status)
	}
	m.visit.Status = *update.Status
	return 1, nil
}

func TestReachedConsentStepHandler(t *testing.T) {
	dataAPI := &mockConsentDataAPI{
		visit: &common.PatientVisit{},
	}
	h := NewReachedConsentStep(dataAPI)
	b, err := json.Marshal(&reachedConsentStepPostRequest{VisitID: 1})
	test.OK(t, err)

	// Make sure the handler validates ownership of the visit

	dataAPI.visit.PatientID = common.NewPatientID(2)
	r, err := http.NewRequest("POST", "/", bytes.NewReader(b))
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RolePatient})
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusForbidden, w.Code)

	// Should succeed

	dataAPI.visit.Status = common.PVStatusOpen
	dataAPI.visit.PatientID = common.NewPatientID(1)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(b))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, common.PVStatusPendingParentalConsent, dataAPI.visit.Status)

	// Request should be idempotent

	dataAPI.visit.Status = common.PVStatusPendingParentalConsent
	dataAPI.visit.PatientID = common.NewPatientID(1)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(b))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, common.PVStatusPendingParentalConsent, dataAPI.visit.Status)
}
