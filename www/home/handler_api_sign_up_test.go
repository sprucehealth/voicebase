package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/www"
)

type mockDataAPI_signUp struct {
	api.DataAPI
}

func (a *mockDataAPI_signUp) RegisterPatient(p *common.Patient) error {
	return nil
}

func (a *mockDataAPI_signUp) AvailableStates() ([]*common.State, error) {
	return []*common.State{{Abbreviation: "CA"}}, nil
}

type mockAuthAPI_signUp struct {
	api.AuthAPI
	accounts map[string]*mockAccount
}

func (a *mockAuthAPI_signUp) Authenticate(email, password string) (*common.Account, error) {
	acc := a.accounts[email]
	if acc == nil {
		return nil, api.ErrLoginDoesNotExist
	}
	if password != acc.password {
		return nil, api.ErrInvalidPassword
	}
	return acc.account, nil
}

func (a *mockAuthAPI_signUp) CreateToken(accountID int64, platform api.Platform, opt api.CreateTokenOption) (string, error) {
	return "token", nil
}

func (a *mockAuthAPI_signUp) CreateAccount(email, password, role string) (int64, error) {
	if a := a.accounts[email]; a != nil {
		return 0, api.ErrLoginAlreadyExists
	}
	return 1, nil
}

func TestAPISignUpHandler(t *testing.T) {
	dataAPI := &mockDataAPI_signUp{}
	authAPI := &mockAuthAPI_signUp{
		accounts: map[string]*mockAccount{
			"patient@example.com": &mockAccount{
				password: "patient",
				account:  &common.Account{Role: api.RolePatient},
			},
			"doctor@example.com": &mockAccount{
				password: "doctor",
				account:  &common.Account{Role: api.RoleDoctor},
			},
		},
	}
	h := newSignUpAPIHandler(dataAPI, authAPI)

	// Test success

	body, err := json.Marshal(&signUpAPIRequest{
		Email:       "newpatient@example.com",
		Password:    "newpatient",
		State:       "CA",
		FirstName:   "First",
		LastName:    "Last",
		DOB:         encoding.Date{Year: 1970, Month: 1, Day: 1},
		Gender:      "female",
		MobilePhone: common.Phone("415-555-1212"),
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, "{}\n", w.Body.String())
	test.Equals(t, "at=token; Path=/; HttpOnly; Secure", w.Header().Get("Set-Cookie"))

	// Test non-existant state

	body, err = json.Marshal(&signUpAPIRequest{
		Email:       "patient@example.com",
		Password:    "patient",
		State:       "ZZ",
		FirstName:   "First",
		LastName:    "Last",
		DOB:         encoding.Date{Year: 1970, Month: 1, Day: 1},
		Gender:      "female",
		MobilePhone: common.Phone("415-555-1212"),
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, www.HTTPStatusAPIError, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"invalid_request\",\"message\":\"A valid US state is required\"}}\n", w.Body.String())
	test.Equals(t, "", w.Header().Get("Set-Cookie"))

	// Test existing account email

	body, err = json.Marshal(&signUpAPIRequest{
		Email:       "patient@example.com",
		Password:    "patient",
		State:       "CA",
		FirstName:   "First",
		LastName:    "Last",
		DOB:         encoding.Date{Year: 1970, Month: 1, Day: 1},
		Gender:      "female",
		MobilePhone: common.Phone("415-555-1212"),
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, www.HTTPStatusAPIError, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"account_exists\",\"message\":\"An account already exists with the provided email address\"}}\n", w.Body.String())
	test.Equals(t, "", w.Header().Get("Set-Cookie"))

	// Test invalid email

	body, err = json.Marshal(&signUpAPIRequest{
		Email:       "newpatient@...",
		Password:    "newpatient",
		State:       "CA",
		FirstName:   "First",
		LastName:    "Last",
		DOB:         encoding.Date{Year: 1970, Month: 1, Day: 1},
		Gender:      "female",
		MobilePhone: common.Phone("415-555-1212"),
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, www.HTTPStatusAPIError, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"invalid_request\",\"message\":\"The email provided is invalid\"}}\n", w.Body.String())
	test.Equals(t, "", w.Header().Get("Set-Cookie"))

	// Test invalid phone number

	body, err = json.Marshal(&signUpAPIRequest{
		Email:       "newpatient@example.com",
		Password:    "newpatient",
		State:       "CA",
		FirstName:   "First",
		LastName:    "Last",
		DOB:         encoding.Date{Year: 1970, Month: 1, Day: 1},
		Gender:      "female",
		MobilePhone: common.Phone("1212"),
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, http.StatusBadRequest, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"bad_request\",\"message\":\"Phone number has to be atleast 10 digits long\"}}\n", w.Body.String())
	test.Equals(t, "", w.Header().Get("Set-Cookie"))

	// Test invalid dob (< 18)

	body, err = json.Marshal(&signUpAPIRequest{
		Email:       "newpatient@example.com",
		Password:    "newpatient",
		State:       "CA",
		FirstName:   "First",
		LastName:    "Last",
		DOB:         encoding.Date{Year: 2015, Month: 1, Day: 1},
		Gender:      "female",
		MobilePhone: common.Phone("415-555-1212"),
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, www.HTTPStatusAPIError, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"invalid_request\",\"message\":\"Must be over 18 or over\"}}\n", w.Body.String())
	test.Equals(t, "", w.Header().Get("Set-Cookie"))
}
