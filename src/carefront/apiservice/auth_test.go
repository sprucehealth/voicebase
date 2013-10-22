package apiservice

import (
	"carefront/mockapi"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"encoding/json"
)

const (
	SignupPath = "/v1/signup"
	LoginPath  = "/v1/authenticate"
	LogoutPath = "/v1/logout"
)

// TESTS

func TestSuccesfulSignup(t *testing.T) {
	mux := setupAuthHandlerInMux(SignupPath)
	req, _ := http.NewRequest("POST", SignupPath, strings.NewReader("login=kkjj&password=12345"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusOK, responseWriter, t)
	validateTokenResponse(responseWriter.body, t)
}

func TestExistingUserInSignup(t *testing.T) {
	mux := setupAuthHandlerInMux(SignupPath)
	req, _ := http.NewRequest("POST", SignupPath, strings.NewReader("login=kajham&password=12345"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusBadRequest, responseWriter, t) 
}

func TestMissingParametersSignup(t *testing.T) {
	mux := setupAuthHandlerInMux(SignupPath)
	req, _ := http.NewRequest("POST", SignupPath, nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusForbidden, responseWriter, t)
}

func TestSignupFollowedByLogin(t *testing.T) {
	mux := setupAuthHandlerInMux(SignupPath)
	req, _ := http.NewRequest("POST", SignupPath, strings.NewReader("login=kjkj&password=12345"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusOK, responseWriter, t)
	validateTokenResponse(responseWriter.body, t)

	anotherAuthHandler := &AuthenticationHandler{mux.AuthApi}
	mux.Handle(LoginPath, anotherAuthHandler)
	req, _ = http.NewRequest("POST", LoginPath, strings.NewReader("login=kjkj&password=12345"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter = &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusOK, responseWriter, t )
	validateTokenResponse(responseWriter.body, t)
}

func TestSuccessfulLogin(t *testing.T) {
	mux := setupAuthHandlerInMux(LoginPath)

	req, _ := http.NewRequest("POST", LoginPath, strings.NewReader("login=kajham&password=12345"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(http.StatusOK) {
		t.Errorf("Expected status code %d, but got %q", http.StatusOK, statusCode)
	}
	validateTokenResponse(responseWriter.body, t)
}

func TestUnsuccessfulLoginDueToPassword(t *testing.T) {
	mux := setupAuthHandlerInMux(LoginPath)

	req, _ := http.NewRequest("POST", LoginPath, strings.NewReader("login=kajham&password=ShouldFail"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(http.StatusForbidden) {
		t.Errorf("Expected status code %d, but got %q", http.StatusForbidden, statusCode)
	}
}

func TestUnsuccessfulLoginDueToUsername(t *testing.T) {
	mux := setupAuthHandlerInMux(LoginPath)

	req, _ := http.NewRequest("POST", LoginPath, strings.NewReader("login=kajaja&password=12345"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(http.StatusForbidden) {
		t.Errorf("Expected status code %d, but got %q", http.StatusForbidden, statusCode)
	}
}

func TestUnsuccessfulLoginDueToMissingParams(t *testing.T) {
	mux := setupAuthHandlerInMux(LoginPath)

	req, _ := http.NewRequest("POST", LoginPath, nil)

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

func setupAuthHandlerInMux(path string) *AuthServeMux {
	fakeAuthApi := createAndReturnFakeAuthApi()
	authHandler := &AuthenticationHandler{fakeAuthApi}
	mux := &AuthServeMux{*http.NewServeMux(), fakeAuthApi}
	mux.Handle(path, authHandler)

	return mux
}


func validateTokenResponse(data []byte, t *testing.T) {	
	type TokenJson struct {
		Token string
	}

	// test body
	var tokenJson TokenJson
	err := json.Unmarshal(data, &tokenJson)
	if err != nil {
		t.Errorf("Expected an auth token to be returned as response to the login called. %s", err.Error())
	}
	if tokenJson.Token == "" {
		t.Errorf("token not expected to be empty in return!")
	}
}

func checkStatusCode(expected int, responseWriter *FakeResponseWriter, t *testing.T) {
	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(expected) {
		t.Errorf("Expected status code %d, but got %q", expected, statusCode)
	}
}
