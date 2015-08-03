package home

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/medrecord"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/www"
)

type mockDataAPI_parentalMedicalRecord struct {
	api.DataAPI
	consent []*common.ParentalConsent
	patient *common.Patient
}

func (m *mockDataAPI_parentalMedicalRecord) GetPatientIDFromAccountID(accountID int64) (int64, error) {
	return accountID, nil
}

func (m *mockDataAPI_parentalMedicalRecord) ParentalConsent(childPatientID int64) ([]*common.ParentalConsent, error) {
	return m.consent, nil
}

func (m *mockDataAPI_parentalMedicalRecord) Patient(id int64, basic bool) (*common.Patient, error) {
	return m.patient, nil
}

type medRecordRenderer struct{}

func (r *medRecordRenderer) Render(p *common.Patient, opt medrecord.RenderOption) ([]byte, error) {
	return nil, nil
}

func TestParentalMedicalRecordHandler(t *testing.T) {
	dataAPI := &mockDataAPI_parentalMedicalRecord{
		patient: &common.Patient{
			HasParentalConsent: true,
		},
	}
	renderer := &medRecordRenderer{}
	h := newParentalMedicalRecordHandler(dataAPI, renderer)
	account := &common.Account{
		ID:   1,
		Role: api.RolePatient,
	}
	ctx := www.CtxWithAccount(context.Background(), account)
	ctx = mux.SetVars(ctx, map[string]string{"childid": "2"})

	// Parent does not have access to view this child's record

	r, err := http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusNotFound, w)

	// Parent does have access

	dataAPI.consent = []*common.ParentalConsent{{ParentPatientID: 1}}
	r, err = http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)

	// Parent has not yet completed flow

	dataAPI.patient.HasParentalConsent = false
	dataAPI.consent = []*common.ParentalConsent{{ParentPatientID: 1}}
	r, err = http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusSeeOther, w)
}
