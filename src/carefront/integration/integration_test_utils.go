package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"carefront/api"
	"carefront/apiservice"
	"carefront/common/config"
	"carefront/services/auth"
	thriftapi "carefront/thrift/api"
	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
)

var (
	CannotRunTestLocally = errors.New("test: The test database is not set. Skipping test")
)

type TestDBConfig struct {
	User     string
	Password string
	Host     string
}

type TestConf struct {
	DB TestDBConfig `group:"Database" toml:"database"`
}

type TestData struct {
	DataApi             api.DataAPI
	AuthApi             thriftapi.Auth
	DBConfig            *TestDBConfig
	CloudStorageService api.CloudStorageAPI
	DB                  *sql.DB
}

func GetDBConfig(t *testing.T) *TestDBConfig {
	dbConfig := TestConf{}
	fileContents, err := ioutil.ReadFile("../apps/restapi/dev.conf")
	if err != nil {
		t.Fatal("Unable to upload dev.conf to read database data from")
	}
	_, err = toml.Decode(string(fileContents), &dbConfig)
	if err != nil {
		t.Fatal("Error decoding toml data :" + err.Error())
	}
	return &dbConfig.DB
}

func ConnectToDB(t *testing.T, dbConfig *TestDBConfig) *sql.DB {
	databaseName := os.Getenv("TEST_DB")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbConfig.User, dbConfig.Password, dbConfig.Host, databaseName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("Unable to connect to the database" + err.Error())
	}

	err = db.Ping()
	if err != nil {
		t.Fatal("Unable to ping database " + err.Error())
	}
	return db
}

func CheckIfRunningLocally(t *testing.T) error {
	// if the TEST_DB is not set in the environment, we assume
	// that we are running these tests locally, in which case
	// we exit the tests with a warning
	if os.Getenv("TEST_DB") == "" {
		t.Log("WARNING: The test database is not set. Skipping test ")
		return CannotRunTestLocally
	}
	return nil
}

func SetupIntegrationTest(t *testing.T) TestData {
	dbConfig := GetDBConfig(t)
	db := ConnectToDB(t, dbConfig)

	conf := config.BaseConfig{}
	awsAuth, err := conf.AWSAuth()
	if err != nil {
		t.Fatal("Error trying to get auth setup: " + err.Error())
	}
	cloudStorageService := api.NewCloudStorageService(awsAuth)

	authApi := &auth.AuthService{DB: db}
	dataApi := &api.DataService{DB: db}

	testData := TestData{DataApi: dataApi,
		AuthApi:             authApi,
		DBConfig:            dbConfig,
		CloudStorageService: cloudStorageService,
		DB:                  db,
	}

	return testData
}

func CheckSuccessfulStatusCode(resp *http.Response, errorMessage string, t *testing.T) {
	if resp.StatusCode != http.StatusOK {
		t.Fatal(errorMessage)
	}
}

func SignupRandomTestPatient(t *testing.T, dataApi api.DataAPI, authApi thriftapi.Auth) *apiservice.PatientSignedupResponse {
	authHandler := &apiservice.SignupPatientHandler{AuthApi: authApi, DataApi: dataApi}
	ts := httptest.NewServer(authHandler)
	defer ts.Close()

	requestBody := bytes.NewBufferString("first_name=Test&last_name=Test&email=")
	requestBody.WriteString(strconv.FormatInt(time.Now().Unix(), 10))
	requestBody.WriteString("@example.com&password=12345&dob=11/08/1987&zip_code=94115&gender=male")
	res, err := http.Post(ts.URL, "application/x-www-form-urlencoded", requestBody)
	if err != nil {
		t.Fatal("Unable to make post request for registering patient: " + err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}
	CheckSuccessfulStatusCode(res, fmt.Sprintf("Unable to make success request to signup patient. Here's the code returned %d and here's the body of the request %s", res.StatusCode, body), t)

	signedupPatientResponse := &apiservice.PatientSignedupResponse{}
	err = json.Unmarshal(body, signedupPatientResponse)
	if err != nil {
		t.Fatal("Unable to parse response from patient signed up")
	}
	return signedupPatientResponse
}

func GetPatientVisitForPatient(PatientId int64, testData TestData, t *testing.T) *apiservice.PatientVisitResponse {
	patientVisitHandler := apiservice.NewPatientVisitHandler(testData.DataApi, testData.AuthApi,
		testData.CloudStorageService, testData.CloudStorageService)
	patient, err := testData.DataApi.GetPatientFromId(PatientId)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	patientVisitHandler.AccountIdFromAuthToken(patient.AccountId)
	ts := httptest.NewServer(patientVisitHandler)
	defer ts.Close()

	// register a patient visit for this patient
	client := &http.Client{}
	req, _ := http.NewRequest("GET", ts.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the response for the new patient visit call: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call to register new patient visit: "+string(body), t)

	patientVisitResponse := &apiservice.PatientVisitResponse{}
	err = json.Unmarshal(body, patientVisitResponse)
	if err != nil {
		t.Fatal("Unable to unmarshall response body into patient visit response: " + err.Error())
	}

	return patientVisitResponse
}
