package apiservice

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
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
	var ctx *Context
	apiStub := &authAPIStub{}
	h := NoAuthenticationRequiredHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = GetContext(r)
		w.WriteHeader(http.StatusAccepted)
	}), apiStub)

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected %d, got %d", http.StatusAccepted, w.Code)
	}
	if ctx == nil {
		t.Fatal("Context not set")
	}
	if ctx.Role != "" || ctx.AccountID != 0 {
		t.Fatal("Expected empty context")
	}

	// Make sure the request is authenticated if a valid token is included
	ctx = nil
	r, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "token abc")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected %d, got %d", http.StatusAccepted, w.Code)
	}
	if ctx == nil {
		t.Fatal("Context not set")
	}
	if ctx.Role != api.RolePatient || ctx.AccountID != 1 {
		t.Fatalf("Expected role PATIENT and account ID 1, got %+v", ctx)
	}
}

func TestAuthenticationRequiredHandler(t *testing.T) {
	var ctx *Context
	apiStub := &authAPIStub{}
	h := AuthenticationRequiredHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = GetContext(r)
		w.WriteHeader(http.StatusAccepted)
	}), apiStub)

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected %d, got %d", http.StatusAccepted, w.Code)
	}
	// Make sure handler isn't called
	if ctx != nil {
		t.Fatal("Context should not have been set")
	}

	// Make sure the request is authenticated if a valid token is included
	r, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "token abc")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected %d, got %d", http.StatusAccepted, w.Code)
	}
	if ctx == nil {
		t.Fatal("Context not set")
	}
	if ctx.Role != api.RolePatient || ctx.AccountID != 1 {
		t.Fatalf("Expected role PATIENT and account ID 1, got %+v", ctx)
	}
}
