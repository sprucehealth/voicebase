package notifications

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/common/config"
	"carefront/libs/aws/sns"
	"carefront/notify"
	patientApi "carefront/patient"
	"carefront/test/test_integration"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// Test registering device token for first time
func TestRegisteringToken_Patient(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	accountId := patient.AccountId.Int64()

	deviceToken := "12345"
	notificationConfigs := map[string]*config.NotificationConfig{
		"iOS-Patient-Feature": &config.NotificationConfig{
			SNSApplicationEndpoint: "endpoint",
		},
	}
	mockSNSClient := &sns.MockSNS{
		PushEndpointToReturn: "push_endpoint",
	}

	SetDeviceTokenForAccountId(accountId, deviceToken, notificationConfigs, mockSNSClient, testData.DataApi, t)
	// get user communication item and ensure that its all setup
}

func TestRegisteringToken_Doctor(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err.Error())
	}
	accountId := doctor.AccountId.Int64()

	deviceToken := "12345"
	notificationConfigs := map[string]*config.NotificationConfig{
		"iOS-Patient-Feature": &config.NotificationConfig{
			SNSApplicationEndpoint: "endpoint",
		},
	}
	mockSNSClient := &sns.MockSNS{
		PushEndpointToReturn: "push_endpoint",
	}

	SetDeviceTokenForAccountId(accountId, deviceToken, notificationConfigs, mockSNSClient, testData.DataApi, t)
	// get user communication item and ensure that its all setup
}

func TestRegisteringToken_SameToken(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err.Error())
	}
	accountId := doctor.AccountId.Int64()

	deviceToken := "12345"
	notificationConfigs := map[string]*config.NotificationConfig{
		"iOS-Patient-Feature": &config.NotificationConfig{
			SNSApplicationEndpoint: "endpoint",
		},
	}
	mockSNSClient := &sns.MockSNS{
		PushEndpointToReturn: "push_endpoint",
	}

	SetDeviceTokenForAccountId(accountId, deviceToken, notificationConfigs, mockSNSClient, testData.DataApi, t)
	SetDeviceTokenForAccountId(accountId, deviceToken, notificationConfigs, mockSNSClient, testData.DataApi, t)
	if pushConfigDatas, err := testData.DataApi.GetPushConfigDataForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(pushConfigDatas) != 1 {
		t.Fatalf("Expected 1 item instead got %d", len(pushConfigDatas))
	}
}

func TestRegisteringToken_SameTokenDifferentUser(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	accountId := patient.AccountId.Int64()

	deviceToken := "12345"
	notificationConfigs := map[string]*config.NotificationConfig{
		"iOS-Patient-Feature": &config.NotificationConfig{
			SNSApplicationEndpoint: "endpoint",
		},
	}
	mockSNSClient := &sns.MockSNS{
		PushEndpointToReturn: "push_endpoint",
	}

	SetDeviceTokenForAccountId(accountId, deviceToken, notificationConfigs, mockSNSClient, testData.DataApi, t)

	// new patient
	pr = test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient = pr.Patient
	accountId2 := patient.AccountId.Int64()

	SetDeviceTokenForAccountId(accountId2, deviceToken, notificationConfigs, mockSNSClient, testData.DataApi, t)
	if pushConfigDatas, err := testData.DataApi.GetPushConfigDataForAccount(accountId2); err != nil {
		t.Fatalf(err.Error())
	} else if len(pushConfigDatas) != 1 {
		t.Fatalf("Expected 1 item instead got %d", len(pushConfigDatas))
	}

	// older patient should have no tokens anymore
	if pushConfigDatas, err := testData.DataApi.GetPushConfigDataForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(pushConfigDatas) != 0 {
		t.Fatalf("Expected 0 item instead got %d", len(pushConfigDatas))
	}

	if communicationPreferences, err := testData.DataApi.GetCommunicationPreferencesForAccount(accountId); err != nil {
		t.Fatal(err.Error())
	} else if len(communicationPreferences) != 0 {
		t.Fatalf("Expected 0 items instead got %d", len(communicationPreferences))
	}
}

func TestRegisteringToken_DifferentToken(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err.Error())
	}
	accountId := doctor.AccountId.Int64()

	notificationConfigs := map[string]*config.NotificationConfig{
		"iOS-Patient-Feature": &config.NotificationConfig{
			SNSApplicationEndpoint: "endpoint",
		},
	}
	mockSNSClient := &sns.MockSNS{
		PushEndpointToReturn: "push_endpoint",
	}

	SetDeviceTokenForAccountId(accountId, "12345", notificationConfigs, mockSNSClient, testData.DataApi, t)
	SetDeviceTokenForAccountId(accountId, "123456789", notificationConfigs, mockSNSClient, testData.DataApi, t)
	if pushConfigDatas, err := testData.DataApi.GetPushConfigDataForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(pushConfigDatas) != 2 {
		t.Fatalf("Expected 1 item instead got %d", len(pushConfigDatas))
	}
}

func TestRegisteringToken_DeleteOnLogout(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	accountId := patient.AccountId.Int64()

	deviceToken := "12345"
	notificationConfigs := map[string]*config.NotificationConfig{
		"iOS-Patient-Feature": &config.NotificationConfig{
			SNSApplicationEndpoint: "endpoint",
		},
	}
	mockSNSClient := &sns.MockSNS{
		PushEndpointToReturn: "push_endpoint",
	}

	SetDeviceTokenForAccountId(accountId, deviceToken, notificationConfigs, mockSNSClient, testData.DataApi, t)
	SetDeviceTokenForAccountId(accountId, "123456789", notificationConfigs, mockSNSClient, testData.DataApi, t)

	// log the user out
	authHandler := patientApi.NewAuthenticationHandler(testData.DataApi, testData.AuthApi, nil, "")
	authServer := httptest.NewServer(authHandler)
	request, err := http.NewRequest("POST", authServer.URL+"/v1/logout", nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	request.Header.Set("Authorization", "token "+pr.Token)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf(err.Error())
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d instead", http.StatusOK, res.StatusCode)
	}

	// there should be no push communication preference or push config data for this patient
	if pushConfigDatas, err := testData.DataApi.GetPushConfigDataForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(pushConfigDatas) != 0 {
		t.Fatalf("Expected 0 item instead got %d", len(pushConfigDatas))
	}
	if communicationPreferences, err := testData.DataApi.GetCommunicationPreferencesForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(communicationPreferences) != 0 {
		t.Fatalf("Expected 0 communication preference instead got %d", len(communicationPreferences))
	}
}

// Test using appropriate notification config based on spruce header
func TestRegisteringToken_NoConfig(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err.Error())
	}
	accountId := doctor.AccountId.Int64()

	deviceToken := "12345"
	notificationConfigs := map[string]*config.NotificationConfig{}
	mockSNSClient := &sns.MockSNS{
		PushEndpointToReturn: "push_endpoint",
	}

	params := url.Values{}
	params.Set("device_token", deviceToken)

	pNotificationHander := notify.NewNotificationHandler(testData.DataApi, notificationConfigs, mockSNSClient)
	notificationServer := httptest.NewServer(pNotificationHander)
	request, err := http.NewRequest("POST", notificationServer.URL, strings.NewReader(params.Encode()))
	if err != nil {
		t.Fatalf(err.Error())
	}

	request.Header.Set("AccountId", strconv.FormatInt(accountId, 10))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	apiservice.TestingContext.AccountId = accountId
	setupRequestHeaders(request)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf(err.Error())
	} else if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}
}

func setupRequestHeaders(r *http.Request) {
	r.Header.Set("S-Version", `Patient;Feature;0.9.0;000105`)
	r.Header.Set("S-OS", "iOS;7.1.1")
	r.Header.Set("S-Device", "Phone;iPhone6,1;640;1136;2.0")
	r.Header.Set("S-Device-ID", "68753A44-4D6F-1226-9C60-0050E4C00067")
}

func SetDeviceTokenForAccountId(accountId int64, deviceToken string, notificationConfigs map[string]*config.NotificationConfig, mockSNSClient *sns.MockSNS, dataApi api.DataAPI, t *testing.T) {
	params := url.Values{}
	params.Set("device_token", deviceToken)

	pNotificationHander := notify.NewNotificationHandler(dataApi, notificationConfigs, mockSNSClient)
	notificationServer := httptest.NewServer(pNotificationHander)
	request, err := http.NewRequest("POST", notificationServer.URL, strings.NewReader(params.Encode()))
	if err != nil {
		t.Fatalf(err.Error())
	}

	request.Header.Set("AccountId", strconv.FormatInt(accountId, 10))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	apiservice.TestingContext.AccountId = accountId
	setupRequestHeaders(request)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf(err.Error())
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}

	// get push config data and ensure that all values are set
	if pConfigData, err := dataApi.GetPushConfigData(deviceToken); err != nil {
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

	if communicationPreferences, err := dataApi.GetCommunicationPreferencesForAccount(accountId); err != nil {
		t.Fatalf(err.Error())
	} else if len(communicationPreferences) != 1 {
		t.Fatalf("Expected 1 communication preference instead got %d", len(communicationPreferences))
	} else if communicationPreferences[0].CommunicationType != common.Push {
		t.Fatalf("Expected communication type to be PUSH")
	}

}
