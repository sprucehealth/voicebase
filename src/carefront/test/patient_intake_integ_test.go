package test

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"database/sql"
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

func TestPatientRegistration(t *testing.T) {
	dbConfig := TestConf{}
	fileContents, err := ioutil.ReadFile("../server/dev.conf")
	if err != nil {
		t.Fatal("Unable to upload dev.conf to read database data from")
	}
	_, err = toml.Decode(string(fileContents), &dbConfig)
	if err != nil {
		t.Fatal("Error decoding toml data :" + err.Error())
	}

	databaseName := os.Getenv("TEST_DB")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbConfig.DB.User, dbConfig.DB.Password, dbConfig.DB.Host, databaseName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("Unable to connect to the database" + err.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		t.Fatal("Unable to ping database " + err.Error())
	}

	authApi := &api.AuthService{DB: db}
	dataApi := &api.DataService{DB: db}
	authHandler := &apiservice.SignupPatientHandler{AuthApi: authApi, DataApi: dataApi}
	ts := httptest.NewServer(authHandler)
	defer ts.Close()

	requestBody := bytes.NewBufferString("first_name=Test&last_name=Test&email=testxxxyyyy@example.com&password=12345&dob=11/08/1987&zip_code=94115&gender=male")
	res, err := http.Post(ts.URL, "application/x-www-form-urlencoded", requestBody)
	if err != nil {
		t.Fatal("Unable to make post request for registering patient: " + err.Error())
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}
	fmt.Println(string(body))
}
