package integration

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	thriftapi "carefront/thrift/api"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func SignupRandomTestDoctor(t *testing.T, dataApi api.DataAPI, authApi thriftapi.Auth) (signedupDoctorResponse *apiservice.DoctorSignedupResponse, email, password string) {
	authHandler := &apiservice.SignupDoctorHandler{AuthApi: authApi, DataApi: dataApi}
	ts := httptest.NewServer(authHandler)
	defer ts.Close()

	requestBody := bytes.NewBufferString("first_name=Test&last_name=Test&email=")
	email = strconv.FormatInt(time.Now().Unix(), 10) + "@example.com"
	password = "12345"
	requestBody.WriteString(email)
	requestBody.WriteString("&password=")
	requestBody.WriteString(password)
	requestBody.WriteString("&dob=11/08/1987&gender=male")
	res, err := http.Post(ts.URL, "application/x-www-form-urlencoded", requestBody)
	if err != nil {
		t.Fatal("Unable to make post request for registering patient: " + err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}
	CheckSuccessfulStatusCode(res, fmt.Sprintf("Unable to make success request to signup patient. Here's the code returned %d and here's the body of the request %s", res.StatusCode, body), t)

	signedupDoctorResponse = &apiservice.DoctorSignedupResponse{}
	err = json.Unmarshal(body, signedupDoctorResponse)
	if err != nil {
		t.Fatal("Unable to parse response from patient signed up")
	}
	return signedupDoctorResponse, email, password
}
