package notifications

import (
	"carefront/common"
	"carefront/notify"
	"carefront/test/integration"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Test prompt status on login and signup
func TestPromptStatus_Signup(t *testing.T) {
	testData := integration.SetupIntegrationTest(t)
	defer integration.TearDownIntegrationTest(t, testData)

	pr := integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient

	if patient.PromptStatus != common.Unprompted {
		t.Fatalf("Expected prompt status %s but got %s", common.Unprompted, patient.PromptStatus)
	}
}

func TestPromptStatus_Login(t *testing.T) {
	testData := integration.SetupIntegrationTest(t)
	defer integration.TearDownIntegrationTest(t, testData)

	pr := integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient

	// this method would be called when trying to login so checking directly with data service layer
	patient, err := testData.DataApi.GetPatientFromAccountId(patient.AccountId.Int64())
	if err != nil {
		t.Fatalf(err.Error())
	}

	if patient.PromptStatus != common.Unprompted {
		t.Fatalf("Expected prompt status %s but got %s", common.Unprompted, patient.PromptStatus)
	}
}

// Test prompt status after being set
func TestPromptStatus_OnModify(t *testing.T) {
	testData := integration.SetupIntegrationTest(t)
	defer integration.TearDownIntegrationTest(t, testData)

	pr := integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient

	promptStatusHandler := notify.NewPatientPromptStatusHandler(testData.DataApi)
	statusServer := httptest.NewServer(promptStatusHandler)
	params := url.Values{}
	params.Set("prompt_status", "DECLINED")

	res, err := integration.AuthPost(statusServer.URL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()), patient.AccountId.Int64())
	if err != nil {
		t.Fatal(err.Error())
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d instead got %d", http.StatusOK, res.StatusCode)
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal(err.Error())
	}

	if patient.PromptStatus != common.Declined {
		t.Fatalf("Expected prompt status %s instead got %s", common.Declined, patient.PromptStatus)
	}
}
