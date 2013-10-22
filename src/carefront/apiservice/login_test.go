package apiservice

import (
	"carefront/mockapi"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// TESTS

func TestSuccessfulLogin(t *testing.T) {
	mux := setupAuthHandlerInMux()

	req, _ := http.NewRequest("POST", "/v1/authenticate", strings.NewReader("login=kajham&password=12345"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(http.StatusOK) {
		t.Errorf("Expected status code %d, but got %q", http.StatusOK, statusCode)
	}
}

func TestUnsuccessfulLoginDueToPassword(t *testing.T) {
	mux := setupAuthHandlerInMux()

	req, _ := http.NewRequest("POST", "/v1/authenticate", strings.NewReader("login=kajham&password=ShouldFail"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(http.StatusForbidden) {
		t.Errorf("Expected status code %d, but got %q", http.StatusForbidden, statusCode)
	}
}

func TestUnsuccessfulLoginDueToUsername(t *testing.T) {
	mux := setupAuthHandlerInMux()

	req, _ := http.NewRequest("POST", "/v1/authenticate", strings.NewReader("login=kajaja&password=12345"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(http.StatusForbidden) {
		t.Errorf("Expected status code %d, but got %q", http.StatusForbidden, statusCode)
	}
}

func TestUnsuccessfulLoginDueToMissingParams(t *testing.T) {
	mux := setupAuthHandlerInMux()

	req, _ := http.NewRequest("POST", "/v1/authenticate", nil)

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(http.StatusForbidden) {
		t.Errorf("Expected status code %d, but got %q", http.StatusForbidden, statusCode)
	}
}

// Private Methods

func createAndReturnFakeAuthApi() *mockapi.MockAuth {
	return &mockapi.MockAuth{
		Accounts: map[string]mockapi.MockAccount{
			"kajham": mockapi.MockAccount{
				Id:       1,
				Login:    "kajham",
				Password: "12345",
			},
		},
		Tokens: map[string]int64{
			"tokenForKajham": 1,
		},
	}
}


func setupAuthHandlerInMux() *AuthServeMux {
	fakeAuthApi := createAndReturnFakeAuthApi()
	authHandler := &AuthenticationHandler{fakeAuthApi}
	mux := &AuthServeMux{*http.NewServeMux(), fakeAuthApi}
	mux.Handle("/v1/authenticate", authHandler)

	return mux

}
