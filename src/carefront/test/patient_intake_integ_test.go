package test

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
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

func getDBConfig(t *testing.T) *TestDBConfig {
	dbConfig := TestConf{}
	fileContents, err := ioutil.ReadFile("../server/dev.conf")
	if err != nil {
		t.Fatal("Unable to upload dev.conf to read database data from")
	}
	_, err = toml.Decode(string(fileContents), &dbConfig)
	if err != nil {
		t.Fatal("Error decoding toml data :" + err.Error())
	}
	return &dbConfig.DB
}

func connectToDB(t *testing.T, dbConfig *TestDBConfig) *sql.DB {
	databaseName := os.Getenv("TEST_DB")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbConfig.User, dbConfig.Password, dbConfig.Host, databaseName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("Unable to connect to the database" + err.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		t.Fatal("Unable to ping database " + err.Error())
	}
	return db
}

func signupRandomTestPatient(t *testing.T, dataApi api.DataAPI, authApi api.Auth) *apiservice.PatientSignedupResponse {
	authHandler := &apiservice.SignupPatientHandler{AuthApi: authApi, DataApi: dataApi}
	ts := httptest.NewServer(authHandler)
	defer ts.Close()

	requestBody := bytes.NewBufferString("first_name=Test&last_name=Test&email=testxxxyvavayyy@example.com&password=12345&dob=11/08/1987&zip_code=94115&gender=male")
	res, err := http.Post(ts.URL, "application/x-www-form-urlencoded", requestBody)
	if err != nil {
		t.Fatal("Unable to make post request for registering patient: " + err.Error())
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}
	signedupPatientResponse := &apiservice.PatientSignedupResponse{}
	err = json.Unmarshal(body, signedupPatientResponse)
	if err != nil {
		t.Fatal("Unable to parse response from patient signed up")
	}
	return signedupPatientResponse
}

func TestPatientRegistration(t *testing.T) {
	dbConfig := getDBConfig(t)
	db := connectToDB(t, dbConfig)

	authApi := &api.AuthService{DB: db}
	dataApi := &api.DataService{DB: db}
	signedupPatientResponse := signupRandomTestPatient(t, dataApi, authApi)
	_, err := json.Marshal(signedupPatientResponse)
	if err != nil {
		t.Fatal("Unable to marshal response for signing up patient" + err.Error())
	}
}
