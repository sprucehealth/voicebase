package test_home

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/home"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestHomeCardsForUnAuthenticatedUser(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	homeHandler := home.NewHandler(testData.DataApi, testData.AuthApi)
	patientServer := httptest.NewServer(homeHandler)
	defer patientServer.Close()

	getRequest, err := http.NewRequest("GET", patientServer.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	responseData := make(map[string]interface{})
	res, err := http.DefaultClient.Do(getRequest)
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatal(err)
	} else if items := responseData["items"].([]interface{}); len(items) != 4 {
		t.Fatalf("Expected %d items but got %d", 4, len(items))
	}
	res.Body.Close()

	// now lets try with a signed up patient account;
	// should still get 4 items because its essentially same state as above
	pr := test_integration.SignupRandomTestPatient(t, testData)

	res, err = testData.AuthGet(patientServer.URL, pr.Patient.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatal(err)
	} else if items := responseData["items"].([]interface{}); len(items) != 4 {
		t.Fatalf("Expected %d items but got %d", 4, len(items))
	}
	res.Body.Close()
}
