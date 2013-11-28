package test

import (
	"carefront/apiservice"
	"net/http"
	"strconv"
	"testing"
)

// TESTS

func TestSuccessfulPing(t *testing.T) {
	mux := setupPingHandlerInMux()
	req, _ := http.NewRequest("GET", "http://localhost:8080/v1/ping", nil)
	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)
	statusCode := responseWriter.Headers.Get("Status")
	responseBody := string(responseWriter.body)
	if (responseBody != apiservice.Pong) ||
		(statusCode != strconv.Itoa(http.StatusOK)) {
		t.Errorf("Expected %q with status code %d, but got %q with status code %q", apiservice.Pong, http.StatusOK, responseBody, statusCode)
	}
}

func TestIncorrectTokenPing(t *testing.T) {
	// SETUP
	mux := setupPingHandlerInMux()

	// TEST
	req, _ := http.NewRequest("GET", "http://localhost:8080/v1/ping", nil)
	req.Header.Add("Authorization", "token incorrectToken")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(http.StatusOK) {
		t.Errorf("Expected status code %d, but got status code %q", http.StatusForbidden, statusCode)
	}
}

func TestMalformedAuthorizationHeader(t *testing.T) {
	// SETUP
	mux := setupPingHandlerInMux()
	// TEST
	req, _ := http.NewRequest("GET", "http://localhost:8080/v1/ping", nil)
	req.Header.Add("Authorization", "incorrectToken")

	responseWriter := &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
	mux.ServeHTTP(responseWriter, req)

	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(http.StatusOK) {
		t.Errorf("Expected status code %d, but got status code %q", http.StatusForbidden, statusCode)
	}
}

// Private methods

func setupPingHandlerInMux() *apiservice.AuthServeMux {
	fakeAuthApi := createAndReturnFakeAuthApi()
	pingHandler := apiservice.PingHandler(0)
	mux := &apiservice.AuthServeMux{ServeMux: *http.NewServeMux(), AuthApi: fakeAuthApi}
	mux.Handle("/v1/ping", pingHandler)

	return mux
}
