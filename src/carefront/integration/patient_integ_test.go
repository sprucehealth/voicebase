package integration

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/config"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestDBConfig struct {
	User     string
	Password string
	Host     string
}

type TestConf struct {
	DB TestDBConfig `group:"Database" toml:"database"`
}

func TestPatientRegistration(t *testing.T) {
	CheckIfRunningLocally(t)
	dbConfig := GetDBConfig(t)
	db := ConnectToDB(t, dbConfig)
	defer db.Close()

	authApi := &api.AuthService{DB: db}
	dataApi := &api.DataService{DB: db}
	SignupRandomTestPatient(t, dataApi, authApi)
}

func TestPatientVisitCreation(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	dbConfig := GetDBConfig(t)
	db := ConnectToDB(t, dbConfig)
	defer db.Close()

	authApi := &api.AuthService{DB: db}
	dataApi := &api.DataService{DB: db}

	conf := config.BaseConfig{}
	awsAuth, err := conf.AWSAuth()
	if err != nil {
		t.Fatal("Error trying to get auth setup: " + err.Error())
	}
	cloudStorageService := api.NewCloudStorageService(awsAuth)

	signedupPatientResponse := SignupRandomTestPatient(t, dataApi, authApi)
	patientVisitHandler := apiservice.NewPatientVisitHandler(dataApi, authApi, cloudStorageService, cloudStorageService)
	patientVisitHandler.AccountIdFromAuthToken(signedupPatientResponse.PatientId)
	ts := httptest.NewServer(patientVisitHandler)
	defer ts.Close()

	client := &http.Client{}
	req, _ := http.NewRequest("GET", ts.URL, nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Error making request to create patient visit")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unable to make success request to signup patient. Here's the code returned %d and here's the body of the request %s", resp.StatusCode, body)
	}

	patientVisitResponse := &apiservice.PatientVisitResponse{}
	err = json.Unmarshal(body, patientVisitResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response from call to patient visit into the response object: " + err.Error())
	}

	if patientVisitResponse.PatientVisitId == 0 {
		t.Fatal("Patient Visit Id not set when it should be.")
	}

	if patientVisitResponse.ClientLayout == nil {
		t.Fatal("The questions for patient intake should be returned as part of the patient visit")
	}

	// getting the patient visit again as we should get back the same patient visit id
	// since this patient visit has not been completed
	client = &http.Client{}
	req, _ = http.NewRequest("GET", ts.URL, nil)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatal("Error making subsequent patient visit request : " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successful subsequent patient visit request")
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read the body of the response on the subsequent patient visit call: " + err.Error())
	}

	anotherPatientVisitResponse := &apiservice.PatientVisitResponse{}
	err = json.Unmarshal(body, anotherPatientVisitResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body into json response : " + err.Error())
	}

	if anotherPatientVisitResponse.PatientVisitId != patientVisitResponse.PatientVisitId {
		t.Fatal("The patient visit id for subsequent calls should be the same so long as we have not closed/submitted the case")
	}
}
