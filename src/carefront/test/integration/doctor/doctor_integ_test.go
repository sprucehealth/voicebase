package doctor

import (
	"bytes"
	"carefront/apiservice"
	"carefront/test/integration"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoctorRegistration(t *testing.T) {
	if err := integration.CheckIfRunningLocally(t); err == integration.CannotRunTestLocally {
		return
	}

	testData := integration.SetupIntegrationTest(t)
	defer testData.DB.Close()
	SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
}

func TestDoctorAuthentication(t *testing.T) {
	if err := integration.CheckIfRunningLocally(t); err == integration.CannotRunTestLocally {
		return
	}

	testData := integration.SetupIntegrationTest(t)
	defer testData.DB.Close()
	_, email, password := SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)

	doctorAuthHandler := &apiservice.DoctorAuthenticationHandler{testData.AuthApi, testData.DataApi}
	ts := httptest.NewServer(doctorAuthHandler)
	requestBody := bytes.NewBufferString("email=")
	requestBody.WriteString(email)
	requestBody.WriteString("&password=")
	requestBody.WriteString(password)
	res, err := http.Post(ts.URL, "application/x-www-form-urlencoded", requestBody)
	if err != nil {
		t.Fatal("Unable to authenticate doctor " + err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}
	integration.CheckSuccessfulStatusCode(res, fmt.Sprintf("Unable to make success request to authenticate doctor. Here's the code returned %d and here's the body of the request %s", res.StatusCode, body), t)

	authenticatedDoctorResponse := &apiservice.DoctorAuthenticationResponse{}
	err = json.Unmarshal(body, authenticatedDoctorResponse)
	if err != nil {
		t.Fatal("Unable to parse response from patient authenticated")
	}

	if authenticatedDoctorResponse.Token == "" || authenticatedDoctorResponse.DoctorId == 0 {
		t.Fatal("Doctor not authenticated as expected")
	}
}
