package apiservice

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type authAPIStub struct {
	api.AuthAPI
}

func (a *authAPIStub) ValidateToken(token string, platform api.Platform) (*common.Account, error) {
	if token == "abc" {
		return &common.Account{
			ID:   1,
			Role: api.RolePatient,
		}, nil
	}
	return nil, api.ErrTokenDoesNotExist
}

func TestNoAuthenticationRequiredHandler(t *testing.T) {
	var account *common.Account
	apiStub := &authAPIStub{}
	h := NoAuthenticationRequiredHandler(httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		account, _ = CtxAccount(ctx)
		w.WriteHeader(http.StatusAccepted)
	}), apiStub)

	r, err := http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusAccepted, w)
	test.Equals(t, (*common.Account)(nil), account)

	// Make sure the request is authenticated if a valid token is included
	r, err = http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	r.Header.Set("Authorization", "token abc")
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusAccepted, w)
	test.Assert(t, account != nil, "Account not set")
	test.Equals(t, api.RolePatient, account.Role)
	test.Equals(t, int64(1), account.ID)
}

func TestAuthenticationRequiredHandler(t *testing.T) {
	var account *common.Account
	var called bool
	apiStub := &authAPIStub{}
	h := AuthenticationRequiredHandler(httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		account, _ = CtxAccount(ctx)
		called = true
		w.WriteHeader(http.StatusAccepted)
	}), apiStub)

	r, err := http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusForbidden, w)
	// Make sure handler isn't called
	test.Equals(t, false, called)

	// Make sure the request is authenticated if a valid token is included
	r, err = http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	r.Header.Set("Authorization", "token abc")
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusAccepted, w)
	test.Assert(t, account != nil, "Account not set")
	test.Equals(t, api.RolePatient, account.Role)
	test.Equals(t, int64(1), account.ID)
}
