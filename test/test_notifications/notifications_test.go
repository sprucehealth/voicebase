package test_notifications

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test/test_integration"
)

// Test registering device token for first time
func TestRegisteringToken_Patient(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	accountId := patient.AccountId.Int64()

	deviceToken := "12345"

	SetDeviceTokenForAccountId(accountId, deviceToken, testData, t)
}

func TestRegisteringToken_Doctor(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err.Error())
	}
	accountId := doctor.AccountId.Int64()

	deviceToken := "12345"

	SetDeviceTokenForAccountId(accountId, deviceToken, testData, t)
}

func TestRegisteringToken_SameToken(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err.Error())
	}
	accountId := doctor.AccountId.Int64()

	deviceToken := "12345"

	SetDeviceTokenForAccountId(accountId, deviceToken, testData, t)
	SetDeviceTokenForAccountId(accountId, deviceToken, testData, t)
	if pushConfigDataList, err := testData.DataApi.GetPushConfigDataForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(pushConfigDataList) != 1 {
		t.Fatalf("Expected 1 item instead got %d", len(pushConfigDataList))
	}
}

func TestRegisteringToken_SameTokenDifferentUser(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	accountId := patient.AccountId.Int64()

	deviceToken := "12345"
	SetDeviceTokenForAccountId(accountId, deviceToken, testData, t)

	// new patient
	pr = test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient = pr.Patient
	accountId2 := patient.AccountId.Int64()

	SetDeviceTokenForAccountId(accountId2, deviceToken, testData, t)
	if pushConfigDataList, err := testData.DataApi.GetPushConfigDataForAccount(accountId2); err != nil {
		t.Fatalf(err.Error())
	} else if len(pushConfigDataList) != 1 {
		t.Fatalf("Expected 1 item instead got %d", len(pushConfigDataList))
	}

	// older patient should have no tokens anymore
	if pushConfigDataList, err := testData.DataApi.GetPushConfigDataForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(pushConfigDataList) != 0 {
		t.Fatalf("Expected 0 item instead got %d", len(pushConfigDataList))
	}

	if communicationPreferences, err := testData.DataApi.GetCommunicationPreferencesForAccount(accountId); err != nil {
		t.Fatal(err.Error())
	} else if len(communicationPreferences) != 0 {
		t.Fatalf("Expected 0 items instead got %d", len(communicationPreferences))
	}
}

func TestRegisteringToken_DifferentToken(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err.Error())
	}
	accountId := doctor.AccountId.Int64()

	SetDeviceTokenForAccountId(accountId, "12345", testData, t)
	SetDeviceTokenForAccountId(accountId, "123456789", testData, t)
	if pushConfigDataList, err := testData.DataApi.GetPushConfigDataForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(pushConfigDataList) != 2 {
		t.Fatalf("Expected 1 item instead got %d", len(pushConfigDataList))
	}
}

func TestRegisteringToken_DeleteOnLogout(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	accountId := patient.AccountId.Int64()

	deviceToken := "12345"

	SetDeviceTokenForAccountId(accountId, deviceToken, testData, t)
	SetDeviceTokenForAccountId(accountId, "123456789", testData, t)

	// log the user out
	request, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.LogoutURLPath, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	request.Header.Set("Authorization", "token "+pr.Token)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d instead", http.StatusOK, res.StatusCode)
	}

	// there should be no push communication preference or push config data for this patient
	if pushConfigDataList, err := testData.DataApi.GetPushConfigDataForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(pushConfigDataList) != 0 {
		t.Fatalf("Expected 0 item instead got %d", len(pushConfigDataList))
	}
	if communicationPreferences, err := testData.DataApi.GetCommunicationPreferencesForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(communicationPreferences) != 0 {
		t.Fatalf("Expected 0 communication preference instead got %d", len(communicationPreferences))
	}
}

// Test using appropriate notification config based on spruce header
func TestRegisteringToken_NoConfig(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err.Error())
	}
	accountId := doctor.AccountId.Int64()

	deviceToken := "12345"
	params := url.Values{}
	params.Set("device_token", deviceToken)

	request, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.NotificationTokenURLPath, strings.NewReader(params.Encode()))
	if err != nil {
		t.Fatalf(err.Error())
	}

	request.Header.Set("AccountId", strconv.FormatInt(accountId, 10))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setupRequestHeaders(request)
	request.Header.Set("S-Version", `Patient;Demo;0.9.0;000105`)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}
}

func setupRequestHeaders(r *http.Request) {
	r.Header.Set("S-Version", `Patient;Feature;0.9.0;000105`)
	r.Header.Set("S-OS", "iOS;7.1.1")
	r.Header.Set("S-Device", "Phone;iPhone6,1;640;1136;2.0")
	r.Header.Set("S-Device-ID", "68753A44-4D6F-1226-9C60-0050E4C00067")
}

func SetDeviceTokenForAccountId(accountId int64, deviceToken string, testData *test_integration.TestData, t *testing.T) {
	params := url.Values{}
	params.Set("device_token", deviceToken)

	request, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.NotificationTokenURLPath, strings.NewReader(params.Encode()))
	if err != nil {
		t.Fatalf(err.Error())
	}

	request.Header.Set("AccountId", strconv.FormatInt(accountId, 10))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setupRequestHeaders(request)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}

	// get push config data and ensure that all values are set
	if pConfigData, err := testData.DataApi.GetPushConfigData(deviceToken); err != nil {
		t.Fatalf(err.Error())
	} else if pConfigData.Id == 0 {
		t.Fatal("Expected push config data to have an id")
	} else if pConfigData.AccountId == 0 {
		t.Fatal("Expected push config data to have an account id")
	} else if pConfigData.DeviceToken != deviceToken {
		t.Fatalf("Expected device token to be %s instead it was %s", deviceToken, pConfigData.DeviceToken)
	} else if pConfigData.PushEndpoint == "" {
		t.Fatalf("Expected push endpoint to be set")
	} else if pConfigData.Platform == "" {
		t.Fatalf("Expected push endpoint to be set")
	} else if pConfigData.PlatformVersion == "" {
		t.Fatalf("Expected platform version to be set")
	} else if pConfigData.AppType == "" {
		t.Fatalf("Expected app type to be set")
	} else if pConfigData.AppEnvironment == "" {
		t.Fatalf("Expected app environment to be set")
	} else if pConfigData.AppVersion == "" {
		t.Fatalf("Expected app version to be set")
	} else if pConfigData.DeviceModel == "" {
		t.Fatalf("Expected device model to be set")
	} else if pConfigData.Device == "" {
		t.Fatalf("Expected device to be set")
	} else if pConfigData.DeviceID == "" {
		t.Fatalf("Expected deviceID to be set")
	} else if pConfigData.CreationDate.IsZero() {
		t.Fatalf("Expected creation date to be set")
	}

	if communicationPreferences, err := testData.DataApi.GetCommunicationPreferencesForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(communicationPreferences) != 1 {
		t.Fatalf("Expected 1 communication preference instead got %d", len(communicationPreferences))
	} else if communicationPreferences[0].CommunicationType != common.Push {
		t.Fatalf("Expected communication type to be PUSH")
	}

}
