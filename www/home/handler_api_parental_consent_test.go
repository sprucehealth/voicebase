package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type mockDataAPI_parentalConsent struct {
	api.DataAPI
	relationship string
	proof        *api.ParentalConsentProof
	updated      bool
	consent      *common.ParentalConsent
	patients     []*common.Patient
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

func (a *mockDataAPI_parentalConsent) GetPatientIDFromAccountID(accountID int64) (common.PatientID, error) {
	for _, p := range a.patients {
		if p.AccountID.Int64() == accountID {
			return p.ID, nil
		}
	}
	return common.PatientID{}, api.ErrNotFound("patient_id")
}

func (a *mockDataAPI_parentalConsent) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
	for _, p := range a.patients {
		if p.AccountID.Int64() == accountID {
			return p, nil
		}
	}
	return nil, api.ErrNotFound("patient")
}

func (a *mockDataAPI_parentalConsent) GrantParentChildConsent(parentPatientID, childPatientID common.PatientID, relationship string) (bool, error) {
	a.relationship = relationship
	return true, nil
}

func (a *mockDataAPI_parentalConsent) ParentalConsent(childPatientID common.PatientID) ([]*common.ParentalConsent, error) {
	if a.consent == nil {
		return nil, nil
	}
	return []*common.ParentalConsent{a.consent}, nil
}

func (a *mockDataAPI_parentalConsent) ParentConsentProof(parentPatientID common.PatientID) (*api.ParentalConsentProof, error) {
	if a.proof == nil {
		return nil, api.ErrNotFound("proof")
	}
	return a.proof, nil
}

func (a *mockDataAPI_parentalConsent) ParentalConsentCompletedForPatient(patientID common.PatientID) (bool, error) {
	a.updated = true
	return true, nil
}

func (a *mockDataAPI_parentalConsent) Patient(id common.PatientID, basicInfoOnly bool) (*common.Patient, error) {
	for _, p := range a.patients {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, api.ErrNotFound("patient")
}

func TestParentalConsentAPIHandler_POST(t *testing.T) {
	dobOver18 := encoding.Date{Year: 1970}
	dobUnder18 := encoding.Date{Year: time.Now().Year() - 15}
	patients := []*common.Patient{
		// Parent
		{ID: common.NewPatientID(1), AccountID: encoding.DeprecatedNewObjectID(1), DOB: dobOver18},
		// Child
		{ID: common.NewPatientID(2), AccountID: encoding.DeprecatedNewObjectID(2), DOB: dobUnder18},
	}
	dataAPI := &mockDataAPI_parentalConsent{patients: patients}

	h := newParentalConsentAPIHAndler(dataAPI, dispatch.New())

	account := &common.Account{ID: 1, Role: api.RolePatient}
	ctx := www.CtxWithAccount(context.Background(), account)

	// Forbidden

	body, err := json.Marshal(&parentalConsentAPIPOSTRequest{
		ChildPatientID: common.NewPatientID(2),
		Relationship:   "handler",
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	r.AddCookie(newParentalConsentCookie(common.NewPatientID(2), "abc", r))
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusForbidden, w)

	// Success

	*dataAPI = mockDataAPI_parentalConsent{patients: patients}
	body, err = json.Marshal(&parentalConsentAPIPOSTRequest{
		ChildPatientID: common.NewPatientID(2),
		Relationship:   "handler",
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	token, err := patient.GenerateParentalConsentToken(dataAPI, common.NewPatientID(2))
	test.OK(t, err)
	r.AddCookie(newParentalConsentCookie(common.NewPatientID(2), token, r))
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, "handler", dataAPI.relationship)
	test.Equals(t, false, dataAPI.updated)

	// If all steps of consent are complete then patient and visits should be updated

	*dataAPI = mockDataAPI_parentalConsent{
		proof: &api.ParentalConsentProof{
			SelfiePhotoID:       ptr.Int64(111),
			GovernmentIDPhotoID: ptr.Int64(222),
		},
		patients: patients,
	}
	body, err = json.Marshal(&parentalConsentAPIPOSTRequest{
		ChildPatientID: common.NewPatientID(2),
		Relationship:   "handler",
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	token, err = patient.GenerateParentalConsentToken(dataAPI, common.NewPatientID(2))
	test.OK(t, err)
	r.AddCookie(newParentalConsentCookie(common.NewPatientID(2), token, r))
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, "handler", dataAPI.relationship)
	test.Equals(t, true, dataAPI.updated)

	// Disallow underage parent or guardian

	*dataAPI = mockDataAPI_parentalConsent{patients: patients}
	patients[0].DOB = dobUnder18
	body, err = json.Marshal(&parentalConsentAPIPOSTRequest{
		ChildPatientID: common.NewPatientID(2),
		Relationship:   "handler",
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	token, err = patient.GenerateParentalConsentToken(dataAPI, common.NewPatientID(2))
	test.OK(t, err)
	r.AddCookie(newParentalConsentCookie(common.NewPatientID(2), token, r))
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, www.HTTPStatusAPIError, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"under_age\",\"message\":\"A parent or guardian must be 18 or older\"}}\n", w.Body.String())
}

func TestParentalConsentAPIHandler_GET(t *testing.T) {
	dobOver18 := encoding.Date{Year: 1970}
	dobUnder18 := encoding.Date{Year: time.Now().Year() - 15}
	dataAPI := &mockDataAPI_parentalConsent{
		patients: []*common.Patient{
			// Parent
			{ID: common.NewPatientID(1), AccountID: encoding.DeprecatedNewObjectID(1), DOB: dobOver18},
			// Child
			{
				ID:        common.NewPatientID(2),
				AccountID: encoding.DeprecatedNewObjectID(2),
				FirstName: "Timmy",
				LastName:  "Little",
				Gender:    "male",
				DOB:       dobUnder18,
			},
		},
	}

	h := newParentalConsentAPIHAndler(dataAPI, dispatch.New())

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
	token, err := patient.GenerateParentalConsentToken(dataAPI, common.NewPatientID(2))
	test.OK(t, err)
	r.AddCookie(newParentalConsentCookie(common.NewPatientID(2), token, r))
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, "{\"children\":[{\"child_patient_id\":\"2\",\"child_first_name\":\"Timmy\",\"child_gender\":\"male\",\"consented\":false}]}\n", w.Body.String())

	// Access by parent/child link (consented)

	dataAPI.consent = &common.ParentalConsent{
		ParentPatientID: common.NewPatientID(1),
		Consented:       true,
		Relationship:    "someone",
	}
	r, err = http.NewRequest("GET", "/?"+params.Encode(), nil)
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, "{\"children\":[{\"child_patient_id\":\"2\",\"child_first_name\":\"Timmy\",\"child_gender\":\"male\",\"consented\":true,\"relationship\":\"someone\"}]}\n", w.Body.String())
}
