package test_integration

import (
	"bytes"
	"math/rand"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/apiservice/router"
)

func TestPatientSignupInvalidEmail(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)
	testData.StartAPIServer(t)

	email := ".invalid.@email_com"

	requestBody := bytes.NewBufferString("first_name=Test&last_name=Test&email=")
	requestBody.WriteString(strconv.FormatInt(rand.Int63(), 10))
	requestBody.WriteString(email + "&password=12345&dob=1987-11-08&zip_code=94115&phone=7348465522&gender=male")
	res, err := testData.AuthPost(testData.APIServer.URL+router.PatientSignupURLPath, "application/x-www-form-urlencoded", requestBody, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected response code %d. Got %d", http.StatusBadRequest, res.StatusCode)
	}
}
