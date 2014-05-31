package test_integration

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/common/config"
	"carefront/doctor_queue"
	"carefront/homelog"
	"carefront/libs/aws"
	"carefront/libs/dispatch"
	"carefront/notify"
	"carefront/treatment_plan"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	"github.com/samuel/go-metrics/metrics"
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
	AuthApi             api.AuthAPI
	DBConfig            *TestDBConfig
	CloudStorageService api.CloudStorageAPI
	DB                  *sql.DB
	StartTime           time.Time
	AWSAuth             aws.Auth
}

type nullHasher struct{}

func (nullHasher) GenerateFromPassword(password []byte) ([]byte, error) {
	return password, nil
}

func (nullHasher) CompareHashAndPassword(hashedPassword, password []byte) error {
	if !bytes.Equal(hashedPassword, password) {
		return errors.New("Wrong password")
	}
	return nil
}

func init() {
	apiservice.Testing = true
	dispatch.Testing = true
}

func AuthGet(url string, accountId int64) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("AccountId", strconv.FormatInt(accountId, 10))
	apiservice.TestingContext.AccountId = accountId
	return http.DefaultClient.Do(req)
}

func AuthPost(url, bodyType string, body io.Reader, accountId int64) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	req.Header.Set("AccountId", strconv.FormatInt(accountId, 10))
	apiservice.TestingContext.AccountId = accountId
	return http.DefaultClient.Do(req)
}

func AuthPut(url, bodyType string, body io.Reader, accountId int64) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	req.Header.Set("AccountId", strconv.FormatInt(accountId, 10))
	apiservice.TestingContext.AccountId = accountId
	return http.DefaultClient.Do(req)
}

func AuthDelete(url, bodyType string, body io.Reader, accountId int64) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	req.Header.Set("AccountId", strconv.FormatInt(accountId, 10))
	apiservice.TestingContext.AccountId = accountId
	return http.DefaultClient.Do(req)
}

func GetDBConfig(t *testing.T) *TestDBConfig {
	dbConfig := TestConf{}
	fileContents, err := ioutil.ReadFile(os.Getenv(carefrontProjectDirEnv) + "/src/carefront/apps/restapi/dev.conf")
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

func CheckIfRunningLocally(t *testing.T) {
	// if the TEST_DB is not set in the environment, we assume
	// that we are running these tests locally, in which case
	// we exit the tests with a warning
	if os.Getenv(carefrontProjectDirEnv) == "" {
		t.Skip("WARNING: The test database is not set. Skipping test ")
	}
}

func GetDoctorIdOfCurrentPrimaryDoctor(testData TestData, t *testing.T) int64 {
	// get the current primary doctor
	var doctorId int64
	err := testData.DB.QueryRow(`select provider_id from care_provider_state_elligibility 
							inner join role_type on role_type_id = role_type.id 
							inner join care_providing_state on care_providing_state_id = care_providing_state.id
							where role_type_tag='DOCTOR' and care_providing_state.state = 'CA'`).Scan(&doctorId)
	if err != nil {
		t.Fatal("Unable to query for doctor that is elligible to diagnose in CA: " + err.Error())
	}
	return doctorId
}

func SignupAndSubmitPatientVisitForRandomPatient(t *testing.T, testData TestData, doctor *common.Doctor) (*apiservice.PatientVisitResponse, *common.DoctorTreatmentPlan) {
	patientSignedupResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), testData, t)

	patient, err := testData.DataApi.GetPatientFromId(patientSignedupResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}
	answerIntakeRequestBody := prepareAnswersForQuestionsInPatientVisit(patientVisitResponse, t)
	SubmitAnswersIntakeForPatient(patient.PatientId.Int64(), patient.AccountId.Int64(), answerIntakeRequestBody, testData, t)
	SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), patientVisitResponse.PatientVisitId, testData, t)
	// get the patient to start reviewing the case
	StartReviewingPatientVisit(patientVisitResponse.PatientVisitId, doctor, testData, t)
	doctorPickTreatmentPlanResponse := pickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitId, doctor, nil, testData, t)

	return patientVisitResponse, doctorPickTreatmentPlanResponse.TreatmentPlan
}

func SetupIntegrationTest(t *testing.T) TestData {
	CheckIfRunningLocally(t)

	dbConfig := GetDBConfig(t)
	if s := os.Getenv("RDS_INSTANCE"); s != "" {
		dbConfig.Host = s
	}
	if s := os.Getenv("RDS_USERNAME"); s != "" {
		dbConfig.User = s
		dbConfig.Password = os.Getenv("RDS_PASSWORD")
	}

	ts := time.Now()
	setupScript := os.Getenv(carefrontProjectDirEnv) + "/src/carefront/test/test_integration/setup_integration_test.sh"
	cmd := exec.Command(setupScript)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RDS_INSTANCE=%s", dbConfig.Host),
		fmt.Sprintf("RDS_USERNAME=%s", dbConfig.User),
		fmt.Sprintf("RDS_PASSWORD=%s", dbConfig.Password),
	)
	if err := cmd.Run(); err != nil {
		t.Fatal("Unable to run the setup_database.sh script for integration tests: " + err.Error() + " " + out.String())
	}

	dbConfig.DatabaseName = strings.TrimSpace(out.String())
	db := ConnectToDB(t, dbConfig)
	conf := config.BaseConfig{}
	awsAuth, err := conf.AWSAuth()
	if err != nil {
		t.Fatal("Error trying to get auth setup: " + err.Error())
	}
	cloudStorageService := api.NewCloudStorageService(awsAuth)

	authApi := &api.Auth{
		ExpireDuration: time.Minute * 10,
		RenewDuration:  time.Minute * 5,
		DB:             db,
		Hasher:         nullHasher{},
	}
	testData := TestData{
		AuthApi:             authApi,
		DBConfig:            dbConfig,
		CloudStorageService: cloudStorageService,
		DB:                  db,
		AWSAuth:             awsAuth,
	}

	t.Logf("Created and connected to database with name: %s (%.3f seconds)", testData.DBConfig.DatabaseName, float64(time.Since(ts))/float64(time.Second))
	testData.StartTime = time.Now()

	// create the role of a doctor and patient
	_, err = testData.DB.Exec(`insert into role_type (role_type_tag) values ('DOCTOR'),('PATIENT')`)
	if err != nil {
		t.Fatal("Unable to create the provider role of DOCTOR " + err.Error())
	}

	testData.DataApi, err = api.NewDataService(db)
	if err != nil {
		t.Fatalf("Unable to initialize data service layer: %s", err)
	}

	// When setting up the database for each integration test, ensure to setup a doctor that is
	// considered elligible to serve in the state of CA.
	signedupDoctorResponse, _, _ := signupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)

	// make this doctor the primary doctor in the state of CA
	_, err = testData.DB.Exec(`insert into care_provider_state_elligibility (role_type_id, provider_id, care_providing_state_id) 
					values ((select id from role_type where role_type_tag='DOCTOR'), ?, (select id from care_providing_state where state='CA'))`, signedupDoctorResponse.DoctorId)
	if err != nil {
		t.Fatal("Unable to make the signed up doctor the primary doctor elligible in CA to diagnose patients: " + err.Error())
	}

	dispatch.Default = dispatch.New()
	notificationManager := notify.NewManager(testData.DataApi, nil, nil, "", "", "", "", "", nil, metrics.NewRegistry())

	homelog.InitListeners(testData.DataApi, notificationManager)
	treatment_plan.InitListeners(testData.DataApi)
	doctor_queue.InitListeners(testData.DataApi, notificationManager)
	notify.InitListeners(testData.DataApi)

	return testData
}

func TearDownIntegrationTest(t *testing.T, testData TestData) {
	testData.DB.Close()

	t.Logf("Time to run test: %.3f seconds", float64(time.Since(testData.StartTime))/float64(time.Second))
	ts := time.Now()
	// put anything here that is global to the teardown process for integration tests
	teardownScript := os.Getenv(carefrontProjectDirEnv) + "/src/carefront/test/test_integration/teardown_integration_test.sh"
	cmd := exec.Command(teardownScript, testData.DBConfig.DatabaseName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RDS_INSTANCE=%s", testData.DBConfig.Host),
		fmt.Sprintf("RDS_USERNAME=%s", testData.DBConfig.User),
		fmt.Sprintf("RDS_PASSWORD=%s", testData.DBConfig.Password),
	)
	err := cmd.Run()
	if err != nil {
		t.Fatal("Unable to run the teardown integration script for integration tests: " + err.Error() + " " + out.String())
	}
	t.Logf("Tore down database with name: %s (%.3f seconds)", testData.DBConfig.DatabaseName, float64(time.Since(ts))/float64(time.Second))
}

func CheckSuccessfulStatusCode(resp *http.Response, errorMessage string, t *testing.T) {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal(errorMessage + "Response Status " + strconv.Itoa(resp.StatusCode))
	}
}
