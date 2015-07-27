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
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/www"
)

type mockAccount struct {
	password string
	account  *common.Account
}

type mockAuthAPI_signIn struct {
	api.AuthAPI
	accounts map[string]*mockAccount
}

func (a *mockAuthAPI_signIn) Authenticate(email, password string) (*common.Account, error) {
	acc := a.accounts[email]
	if acc == nil {
		return nil, api.ErrLoginDoesNotExist
	}
	if password != acc.password {
		return nil, api.ErrInvalidPassword
	}
	return acc.account, nil
}

func (a *mockAuthAPI_signIn) CreateToken(accountID int64, platform api.Platform, opt api.CreateTokenOption) (string, error) {
	return "token", nil
}

func TestAPISignInHandler(t *testing.T) {
	authAPI := &mockAuthAPI_signIn{
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
	h := newSignInAPIHandler(authAPI)

	// Test success

	body, err := json.Marshal(&signInAPIRequest{
		Email:    "patient@example.com",
		Password: "patient",
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, "{}\n", w.Body.String())
	test.Equals(t, "at=token; Path=/; HttpOnly; Secure", w.Header().Get("Set-Cookie"))

	// Test invalid email

	body, err = json.Marshal(&signInAPIRequest{
		Email:    "nobody@example.com",
		Password: "patient",
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, www.HTTPStatusAPIError, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"invalid_email\",\"message\":\"Invalid email\"}}\n", w.Body.String())
	test.Equals(t, "", w.Header().Get("Set-Cookie"))

	// Test invalid password

	body, err = json.Marshal(&signInAPIRequest{
		Email:    "patient@example.com",
		Password: "nopenopenope",
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, www.HTTPStatusAPIError, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"invalid_password\",\"message\":\"Invalid password\"}}\n", w.Body.String())
	test.Equals(t, "", w.Header().Get("Set-Cookie"))

	// Test not patient role

	body, err = json.Marshal(&signInAPIRequest{
		Email:    "doctor@example.com",
		Password: "doctor",
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, www.HTTPStatusAPIError, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"invalid_role\",\"message\":\"Auth not allowed\"}}\n", w.Body.String())
	test.Equals(t, "", w.Header().Get("Set-Cookie"))
}
