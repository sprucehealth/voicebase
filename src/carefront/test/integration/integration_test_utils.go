package integration

import (
	"bytes"
	"carefront/api"
	"carefront/common/config"
	"carefront/services/auth"
	thriftapi "carefront/thrift/api"
	"database/sql"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	CannotRunTestLocally   = errors.New("test: The test database is not set. Skipping test")
	carefrontProjectDirEnv = "CAREFRONT_PROJECT_DIR"
)

type TestDBConfig struct {
	User         string
	Password     string
	Host         string
	DatabaseName string
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
	fileContents, err := ioutil.ReadFile("../../apps/restapi/dev.conf")
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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.DatabaseName)
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
	if os.Getenv(carefrontProjectDirEnv) == "" {
		t.Log("WARNING: The test database is not set. Skipping test ")
		return CannotRunTestLocally
	}
	return nil
}

func getDoctorIdOfCurrentPrimaryDoctor(testData TestData, t *testing.T) int64 {
	// get the current primary doctor
	var doctorId int64
	err := testData.DB.QueryRow(`select provider_id from care_provider_state_elligibility 
							inner join provider_role on provider_role_id = provider_role.id 
							inner join care_providing_state on care_providing_state_id = care_providing_state.id
							where provider_tag='DOCTOR' and care_providing_state.state = 'CA'`).Scan(&doctorId)
	if err != nil {
		t.Fatal("Unable to query for doctor that is elligible to diagnose in CA: " + err.Error())
	}
	return doctorId
}

func SetupIntegrationTest(t *testing.T) TestData {
	t.Log("Creating new database...")
	ts := time.Now()
	setupScript := os.Getenv(carefrontProjectDirEnv) + "/src/carefront/test/integration/setup_integration_test.sh"
	cmd := exec.Command(setupScript)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		t.Fatal("Unable to run the setup_database.sh script for integration tests: " + err.Error() + " " + out.String())
	}
	fmt.Printf("DEBUG: db build time: %.3f\n", float64(time.Since(ts))/float64(time.Second))

	dbConfig := GetDBConfig(t)
	dbConfig.DatabaseName = strings.TrimSpace(out.String())
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

	t.Log("Created and connected to database with name: " + testData.DBConfig.DatabaseName)

	// When setting up the database for each integration test, ensure to setup a doctor that is
	// considered elligible to serve in the state of CA.
	signedupDoctorResponse, _, _ := SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)

	// create the role of a primary doctor
	_, err = testData.DB.Exec(`insert into provider_role (provider_tag) values ('DOCTOR')`)
	if err != nil {
		t.Fatal("Unable to create the provider role of DOCTOR " + err.Error())
	}

	// make this doctor the primary doctor in the state of CA
	_, err = testData.DB.Exec(`insert into care_provider_state_elligibility (provider_role_id, provider_id, care_providing_state_id) 
					values ((select id from provider_role where provider_tag='DOCTOR'), ?, (select id from care_providing_state where state='CA'))`, signedupDoctorResponse.DoctorId)
	if err != nil {
		t.Fatal("Unable to make the signed up doctor the primary doctor elligible in CA to diagnose patients: " + err.Error())
	}

	return testData
}

func TearDownIntegrationTest(t *testing.T, testData TestData) {
	testData.DB.Close()

	ts := time.Now()
	// put anything here that is global to the teardown process for integration tests
	teardownScript := os.Getenv(carefrontProjectDirEnv) + "/src/carefront/test/integration/teardown_integration_test.sh"
	cmd := exec.Command(teardownScript, testData.DBConfig.DatabaseName)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		t.Fatal("Unable to run the teardown integration script for integration tests: " + err.Error() + " " + out.String())
	}
	t.Log("Tore down database with name: " + testData.DBConfig.DatabaseName)
	fmt.Printf("DEBUG: db teardown time: %.3f\n", float64(time.Since(ts))/float64(time.Second))
}

func CheckSuccessfulStatusCode(resp *http.Response, errorMessage string, t *testing.T) {
	if resp.StatusCode != http.StatusOK {
		t.Fatal(errorMessage + "Response Status " + strconv.Itoa(resp.StatusCode))
	}
}
