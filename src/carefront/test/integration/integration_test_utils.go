package integration

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"carefront/api"
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
	fileContents, err := ioutil.ReadFile("../../../apps/restapi/dev.conf")
	if err != nil {
		t.Fatal("Unable to upload dev.conf to read database data from : " + err.Error())
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
	// running every test that requires database setup parallelly
	// to optimize for speed of tests
	t.Parallel()
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
