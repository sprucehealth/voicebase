package test_integration

import (
	"bytes"
	"carefront/patient"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestPatientSignupInvalidEmail(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	authHandler := patient.NewSignupHandler(testData.DataApi, testData.AuthApi)
	ts := httptest.NewServer(authHandler)
	defer ts.Close()

	email := ".invalid.@email_com"

	requestBody := bytes.NewBufferString("first_name=Test&last_name=Test&email=")
	requestBody.WriteString(strconv.FormatInt(rand.Int63(), 10))
	requestBody.WriteString(email + "&password=12345&dob=1987-11-08&zip_code=94115&phone=7348465522&gender=male")
	res, err := AuthPost(ts.URL, "application/x-www-form-urlencoded", requestBody, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected response code %d. Got %d", http.StatusBadRequest, res.StatusCode)
	}
}
