package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/www"
)

type mockDataAPI_parentalConsent struct {
	api.DataAPI
	relationship string
	proof        *api.ParentalConsentProof
	updated      bool
	consent      *common.ParentalConsent
	patient      *common.Patient
	tokens       map[string]string
}

func (a *mockDataAPI_parentalConsent) CreateToken(purpose, key, token string, expires time.Duration) (string, error) {
	if token == "" {
		token = purpose + key
	}
	if a.tokens == nil {
		a.tokens = make(map[string]string)
	}
	a.tokens[token] = key
	return token, nil
}

func (a *mockDataAPI_parentalConsent) ValidateToken(purpose, token string) (string, error) {
	if k, ok := a.tokens[token]; ok {
		return k, nil
	}
	return "", api.ErrTokenDoesNotExist
}

func (a *mockDataAPI_parentalConsent) GetPatientIDFromAccountID(accountID int64) (int64, error) {
	return accountID, nil
}

func (a *mockDataAPI_parentalConsent) GrantParentChildConsent(parentPatientID, childPatientID int64, relationship string) error {
	a.relationship = relationship
	return nil
}

func (a *mockDataAPI_parentalConsent) ParentalConsent(parentPatientID, childPatientID int64) (*common.ParentalConsent, error) {
	if a.consent == nil {
		return nil, api.ErrNotFound("consent")
	}
	return a.consent, nil
}

func (a *mockDataAPI_parentalConsent) ParentConsentProof(parentPatientID int64) (*api.ParentalConsentProof, error) {
	if a.proof == nil {
		return nil, api.ErrNotFound("proof")
	}
	return a.proof, nil
}

func (a *mockDataAPI_parentalConsent) ParentalConsentCompletedForPatient(patientID int64) error {
	a.updated = true
	return nil
}

func (a *mockDataAPI_parentalConsent) Patient(id int64, basicInfoOnly bool) (*common.Patient, error) {
	return a.patient, nil
}

func TestParentalConsentAPIHandler_POST(t *testing.T) {
	dataAPI := &mockDataAPI_parentalConsent{}

	h := newParentalConsentAPIHAndler(dataAPI)

	account := &common.Account{ID: 1, Role: api.RolePatient}
	ctx := www.CtxWithAccount(context.Background(), account)

	body, err := json.Marshal(&parentalConsentAPIPOSTRequest{
		ChildPatientID: 2,
		Relationship:   "handler",
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	r.AddCookie(newParentalConsentCookie(2, "abc", r))
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusForbidden, w.Code)

	*dataAPI = mockDataAPI_parentalConsent{}
	body, err = json.Marshal(&parentalConsentAPIPOSTRequest{
		ChildPatientID: 2,
		Relationship:   "handler",
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	token, err := patient.GenerateParentalConsentToken(dataAPI, 2)
	test.OK(t, err)
	r.AddCookie(newParentalConsentCookie(2, token, r))
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, "handler", dataAPI.relationship)
	test.Equals(t, false, dataAPI.updated)

	// If all steps of consent are complete then patient and visits should be updated

	*dataAPI = mockDataAPI_parentalConsent{
		proof: &api.ParentalConsentProof{
			SelfiePhotoID:       ptr.Int64(111),
			GovernmentIDPhotoID: ptr.Int64(222),
		},
	}
	body, err = json.Marshal(&parentalConsentAPIPOSTRequest{
		ChildPatientID: 2,
		Relationship:   "handler",
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	token, err = patient.GenerateParentalConsentToken(dataAPI, 2)
	test.OK(t, err)
	r.AddCookie(newParentalConsentCookie(2, token, r))
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, "handler", dataAPI.relationship)
	test.Equals(t, true, dataAPI.updated)
}

func TestParentalConsentAPIHandler_GET(t *testing.T) {
	dataAPI := &mockDataAPI_parentalConsent{
		patient: &common.Patient{
			ID:        encoding.NewObjectID(2),
			FirstName: "Timmy",
			LastName:  "Little",
			Gender:    "male",
		},
	}

	h := newParentalConsentAPIHAndler(dataAPI)

	account := &common.Account{ID: 1, Role: api.RolePatient}
	ctx := www.CtxWithAccount(context.Background(), account)

	// Access denied (no link and no valid token)

	params := url.Values{"child_patient_id": []string{"2"}}
	r, err := http.NewRequest("GET", "/?"+params.Encode(), nil)
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusForbidden, w.Code)

	// Access by token (not consented)

	r, err = http.NewRequest("GET", "/?"+params.Encode(), nil)
	test.OK(t, err)
	token, err := patient.GenerateParentalConsentToken(dataAPI, 2)
	test.OK(t, err)
	r.AddCookie(newParentalConsentCookie(2, token, r))
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, "{\"children\":[{\"child_patient_id\":\"2\",\"child_first_name\":\"Timmy\",\"child_gender\":\"male\",\"consented\":false}]}\n", w.Body.String())

	// Access by parent/child link (consented)

	dataAPI.consent = &common.ParentalConsent{
		Consented:    true,
		Relationship: "someone",
	}
	r, err = http.NewRequest("GET", "/?"+params.Encode(), nil)
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, "{\"children\":[{\"child_patient_id\":\"2\",\"child_first_name\":\"Timmy\",\"child_gender\":\"male\",\"consented\":true,\"relationship\":\"someone\"}]}\n", w.Body.String())
}
