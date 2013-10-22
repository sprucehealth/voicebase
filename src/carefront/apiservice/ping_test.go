package apiservice

import (
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
	if (responseBody != Pong) ||
		(statusCode != strconv.Itoa(http.StatusOK)) {
		t.Errorf("Expected %q with status code %d, but got %q with status code %q", Pong, http.StatusOK, responseBody, statusCode)
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

func setupPingHandlerInMux() *AuthServeMux {
	fakeAuthApi := createAndReturnFakeAuthApi()
	pingHandler := PingHandler(0)
	mux := &AuthServeMux{*http.NewServeMux(), fakeAuthApi}
	mux.Handle("/v1/ping", pingHandler)

	return mux
}
