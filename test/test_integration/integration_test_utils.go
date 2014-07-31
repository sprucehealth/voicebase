package test_integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/payment"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/third_party/github.com/BurntSushi/toml"
	_ "github.com/sprucehealth/backend/third_party/github.com/go-sql-driver/mysql"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

var (
	CannotRunTestLocally = errors.New("test: The test database is not set. Skipping test")
	spruceProjectDirEnv  = "GOPATH"
)

type TestDBConfig struct {
	User         string
	Password     string
	Host         string
	DatabaseName string
}

type TestDosespotConfig struct {
	ClinicId     int64  `long:"clinic_id" description:"Clinic Id for dosespot"`
	ClinicKey    string `long:"clinic_key" description:"Clinic Key for dosespot"`
	UserId       int64  `long:"user_id" description:"User Id for dosespot"`
	SOAPEndpoint string `long:"soap_endpoint" description:"SOAP endpoint"`
	APIEndpoint  string `long:"api_endpoint" description:"API endpoint where soap actions are defined"`
}

type TestConf struct {
	DB       TestDBConfig       `group:"Database" toml:"database"`
	DoseSpot TestDosespotConfig `group:"Dosespot" toml:"dosespot"`
}

type TestData struct {
	DataApi             api.DataAPI
	AuthApi             api.AuthAPI
	ERxAPI              erx.ERxAPI
	DBConfig            *TestDBConfig
	RouterConfig        *router.RouterConfig
	CloudStorageService api.CloudStorageAPI
	DB                  *sql.DB
	AWSAuth             aws.Auth
	APIServer           *httptest.Server
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

func (d *TestData) AuthGet(url string, accountID int64) (*http.Response, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("AccountID", strconv.FormatInt(accountID, 10))

	if accountID > 0 {
		token, err := d.AuthApi.GetToken(accountID)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "token "+token)
	}

	return http.DefaultClient.Do(req)
}

func (d *TestData) AuthPost(url, bodyType string, body io.Reader, accountID int64) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	if accountID > 0 {
		token, err := d.AuthApi.GetToken(accountID)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "token "+token)
	}
	return http.DefaultClient.Do(req)
}

func (d *TestData) AuthPut(url, bodyType string, body io.Reader, accountID int64) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	if accountID > 0 {
		token, err := d.AuthApi.GetToken(accountID)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "token "+token)
	}

	return http.DefaultClient.Do(req)
}

func (d *TestData) AuthDelete(url, bodyType string, body io.Reader, accountID int64) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	if accountID > 0 {
		token, err := d.AuthApi.GetToken(accountID)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "token "+token)
	}

	return http.DefaultClient.Do(req)
}

func GetTestConf(t *testing.T) *TestConf {
	testConf := TestConf{}
	fileContents, err := ioutil.ReadFile(os.Getenv(spruceProjectDirEnv) + "/src/github.com/sprucehealth/backend/test/test.conf")
	if err != nil {
		t.Fatal("Unable to load test.conf to read database data from: " + err.Error())
	}
	_, err = toml.Decode(string(fileContents), &testConf)
	if err != nil {
		t.Fatal("Error decoding toml data :" + err.Error())
	}
	return &testConf

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
	if os.Getenv(spruceProjectDirEnv) == "" {
		t.Skip("WARNING: The test database is not set. Skipping test ")
	}
}

func GetDoctorIdOfCurrentDoctor(testData *TestData, t *testing.T) int64 {
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

func CreateRandomPatientVisitInState(state string, t *testing.T, testData *TestData) *patient_visit.PatientVisitResponse {
	pr := SignupRandomTestPatientInState(state, t, testData)
	pv := CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	answerIntakeRequestBody := PrepareAnswersForQuestionsInPatientVisit(pv, t)
	SubmitAnswersIntakeForPatient(pr.Patient.PatientId.Int64(), pr.Patient.AccountId.Int64(),
		answerIntakeRequestBody, testData, t)
	SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(), pv.PatientVisitId, testData, t)
	return pv
}

func GrantDoctorAccessToPatientCase(t *testing.T, testData *TestData, doctor *common.Doctor, patientCaseId int64) {
	jsonData, err := json.Marshal(&doctor_queue.ClaimPatientCaseRequestData{
		PatientCaseId: encoding.NewObjectId(patientCaseId),
	})

	resp, err := testData.AuthPost(testData.APIServer.URL+router.DoctorCaseClaimURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected response %d instead got %d", http.StatusOK, resp.StatusCode)
	}
}

func AddTestAddressForPatient(patientId int64, testData *TestData, t *testing.T) {
	if err := testData.DataApi.UpdateDefaultAddressForPatient(patientId, &common.Address{
		AddressLine1: "123 Street",
		AddressLine2: "Apt 123",
		City:         "San Francisco",
		State:        "CA",
		ZipCode:      "94115",
	}); err != nil {
		t.Fatal(err)
	}
}

func AddTestPharmacyForPatient(patientId int64, testData *TestData, t *testing.T) {
	if err := testData.DataApi.UpdatePatientPharmacy(patientId, &pharmacy.PharmacyData{
		SourceId:     123,
		PatientId:    patientId,
		Name:         "Test Pharmacy",
		AddressLine1: "123 street",
		AddressLine2: "Suite 250",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}); err != nil {
		t.Fatal(err)
	}
}

func CreateRandomPatientVisitAndPickTP(t *testing.T, testData *TestData, doctor *common.Doctor) (*patient_visit.PatientVisitResponse, *common.DoctorTreatmentPlan) {
	patientSignedupResponse := SignupRandomTestPatient(t, testData)
	patientVisitResponse := CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), testData, t)

	patient, err := testData.DataApi.GetPatientFromId(patientSignedupResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}
	answerIntakeRequestBody := PrepareAnswersForQuestionsInPatientVisit(patientVisitResponse, t)
	SubmitAnswersIntakeForPatient(patient.PatientId.Int64(), patient.AccountId.Int64(), answerIntakeRequestBody, testData, t)
	SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), patientVisitResponse.PatientVisitId, testData, t)
	patientCase, err := testData.DataApi.GetPatientCaseFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}
	GrantDoctorAccessToPatientCase(t, testData, doctor, patientCase.Id.Int64())
	StartReviewingPatientVisit(patientVisitResponse.PatientVisitId, doctor, testData, t)
	doctorPickTreatmentPlanResponse := PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitId, doctor, nil, testData, t)

	return patientVisitResponse, doctorPickTreatmentPlanResponse.TreatmentPlan
}

func GenerateAppEvent(action, resource string, resourceId int64, accountId int64, testData *TestData, t *testing.T) {
	jsonData, err := json.Marshal(&app_event.EventRequestData{
		Resource:   resource,
		ResourceId: resourceId,
		Action:     action,
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := testData.AuthPost(testData.APIServer.URL+router.AppEventURLPath, "application/json", bytes.NewReader(jsonData), accountId)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}
}

func SetupIntegrationTest(t *testing.T) *TestData {
	CheckIfRunningLocally(t)

	testConf := GetTestConf(t)
	dbConfig := &testConf.DB

	if s := os.Getenv("RDS_INSTANCE"); s != "" {
		dbConfig.Host = s
	}
	if s := os.Getenv("RDS_USERNAME"); s != "" {
		dbConfig.User = s
		dbConfig.Password = os.Getenv("RDS_PASSWORD")
	}

	setupScript := os.Getenv(spruceProjectDirEnv) + "/src/github.com/sprucehealth/backend/test/test_integration/setup_integration_test.sh"
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
		t.Fatalf("Unable to run the %s script for integration tests: %s %s", setupScript, err.Error(), out.String())
	}

	dbConfig.DatabaseName = strings.TrimSpace(out.String())
	db := ConnectToDB(t, dbConfig)
	conf := config.BaseConfig{}
	awsAuth, err := conf.AWSAuth()
	if err != nil {
		t.Fatal("Error trying to get auth setup: " + err.Error())
	}

	cloudStorageService := api.NewCloudStorageService(awsAuth)
	erxAPI := erx.NewDoseSpotService(testConf.DoseSpot.ClinicId, testConf.DoseSpot.UserId,
		testConf.DoseSpot.ClinicKey, testConf.DoseSpot.SOAPEndpoint, testConf.DoseSpot.APIEndpoint, nil)
	authApi := &api.Auth{
		ExpireDuration: time.Minute * 10,
		RenewDuration:  time.Minute * 5,
		DB:             db,
		Hasher:         nullHasher{},
	}

	dataAPI, err := api.NewDataService(db)
	if err != nil {
		t.Fatalf("Unable to initialize data service layer: %s", err)
	}

	testData := &TestData{
		DataApi:             dataAPI,
		AuthApi:             authApi,
		ERxAPI:              erxAPI,
		DBConfig:            dbConfig,
		CloudStorageService: cloudStorageService,
		DB:                  db,
		AWSAuth:             awsAuth,
	}

	// create the role of a doctor and patient
	_, err = testData.DB.Exec(`insert into role_type (role_type_tag) values ('DOCTOR'),('PATIENT')`)
	if err != nil {
		t.Fatal("Unable to create the provider role of DOCTOR " + err.Error())
	}

	dispatch.Default = dispatch.New()
	environment.SetCurrent("test")

	testData.RouterConfig = &router.RouterConfig{
		DataAPI:              testData.DataApi,
		AuthAPI:              testData.AuthApi,
		AddressValidationAPI: &address.StubAddressValidationService{},
		PaymentAPI:           &payment.StubPaymentService{},
		NotifyConfigs: (*config.NotificationConfigs)(&map[string]*config.NotificationConfig{
			"iOS-Patient-Feature": &config.NotificationConfig{
				SNSApplicationEndpoint: "endpoint",
			},
		}),
		NotificationManager: notify.NewManager(testData.DataApi, nil, nil, &email.TestService{}, "", "", nil, metrics.NewRegistry()),
		ERxStatusQueue:      &common.SQSQueue{QueueService: &sqs.StubSQS{}},
		ERxAPI:              &erx.StubErxService{},
		SNSClient:           &sns.MockSNS{PushEndpointToReturn: "push_endpoint"},
		MetricsRegistry:     metrics.NewRegistry(),
		CloudStorageAPI:     testData.CloudStorageService,
		ERxRouting:          true,
		APISubdomain:        "api.spruce.local",
		WebSubdomain:        "www.spruce.local",
		EmailService:        &email.TestService{},
	}

	// setup the restapi server
	mux := router.New(testData.RouterConfig)
	testData.APIServer = httptest.NewServer(mux)

	// When setting up the database for each integration test, ensure to setup a doctor that is
	// considered elligible to serve in the state of CA.
	signedupDoctorResponse, _, _ := SignupRandomTestDoctor(t, testData)

	// make this doctor the primary doctor in the state of CA
	careProvidingStateId, err := testData.DataApi.GetCareProvidingStateId("CA", apiservice.HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		t.Fatal(err)
	}

	err = testData.DataApi.MakeDoctorElligibleinCareProvidingState(careProvidingStateId, signedupDoctorResponse.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	return testData
}

func TearDownIntegrationTest(t *testing.T, testData *TestData) {
	testData.DB.Close()
	testData.APIServer.Close()

	// put anything here that is global to the teardown process for integration tests
	teardownScript := os.Getenv(spruceProjectDirEnv) + "/src/github.com/sprucehealth/backend/test/test_integration/teardown_integration_test.sh"
	cmd := exec.Command(teardownScript, testData.DBConfig.DatabaseName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RDS_INSTANCE=%s", testData.DBConfig.Host),
		fmt.Sprintf("RDS_USERNAME=%s", testData.DBConfig.User),
		fmt.Sprintf("RDS_PASSWORD=%s", testData.DBConfig.Password),
	)
	err := cmd.Run()
	if err != nil {
		t.Fatal("Unable to run the teardown integration script for integration tests: " + err.Error() + " " + out.String())
	}
}

func CheckSuccessfulStatusCode(resp *http.Response, errorMessage string, t *testing.T) {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("%s Response Status %d: %s", errorMessage, resp.StatusCode, string(b))
	}
}

func GetAnswerIntakesFromAnswers(aList []common.Answer, t *testing.T) []*common.AnswerIntake {
	answers := make([]*common.AnswerIntake, len(aList))
	for i, a := range aList {
		answers[i] = GetAnswerIntakeFromAnswer(a, t)
	}
	return answers
}

func GetAnswerIntakeFromAnswer(a common.Answer, t *testing.T) *common.AnswerIntake {
	answer, ok := a.(*common.AnswerIntake)
	if !ok {
		t.Fatalf("Expected type AnswerIntake instead got %T", a)
	}
	return answer
}

func GetPhotoIntakeSectionFromAnswer(a common.Answer, t *testing.T) *common.PhotoIntakeSection {
	answer, ok := a.(*common.PhotoIntakeSection)
	if !ok {
		t.Fatalf("Expected type PhotoIntakeSection instead got %T", a)
	}
	return answer
}

func JSONPOSTRequest(t *testing.T, path string, v interface{}) *http.Request {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(v); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}
