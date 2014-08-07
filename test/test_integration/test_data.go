package test_integration

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
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
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/payment"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func init() {
	apiservice.Testing = true
	dispatch.Testing = true
	// To avoid the noise of info logs, lets only display
	// logs that are warning or higher
	golog.Default().SetLevel(golog.WARN)
}

type TestData struct {
	T                   *testing.T
	DataApi             api.DataAPI
	AuthApi             api.AuthAPI
	ERxApi              erx.ERxAPI
	DBConfig            *TestDBConfig
	Config              *router.Config
	CloudStorageService api.CloudStorageAPI
	DB                  *sql.DB
	AWSAuth             aws.Auth
	APIServer           *httptest.Server
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

func (d *TestData) StartAPIServer(t *testing.T) {
	// close any previous api server
	if d.APIServer != nil {
		d.APIServer.Close()
	}

	// setup the restapi server
	mux := router.New(d.Config)
	d.APIServer = httptest.NewServer(mux)

	// FIX: We shouldn't have to signup this doctor, but currently
	// tests expect a default doctor to exist. Probably should get rid of this and update
	// tests to instantiate a doctor if one is needed
	SignupRandomTestDoctorInState("CA", t, d)
}

func (td *TestData) Close() {
	td.DB.Close()

	if td.APIServer != nil {
		td.APIServer.Close()
	}

	// put anything here that is global to the teardown process for integration tests
	teardownScript := os.Getenv(spruceProjectDirEnv) + "/src/github.com/sprucehealth/backend/test/test_integration/teardown_integration_test.sh"
	cmd := exec.Command(teardownScript, td.DBConfig.DatabaseName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RDS_INSTANCE=%s", td.DBConfig.Host),
		fmt.Sprintf("RDS_USERNAME=%s", td.DBConfig.User),
		fmt.Sprintf("RDS_PASSWORD=%s", td.DBConfig.Password),
	)
	err := cmd.Run()
	test.OK(td.T, err)
}

func SetupTest(t *testing.T) *TestData {
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
		DBConfig:            dbConfig,
		CloudStorageService: cloudStorageService,
		DB:                  db,
		AWSAuth:             awsAuth,
		ERxApi: erx.NewDoseSpotService(testConf.DoseSpot.ClinicId, testConf.DoseSpot.UserId,
			testConf.DoseSpot.ClinicKey, testConf.DoseSpot.SOAPEndpoint, testConf.DoseSpot.APIEndpoint, nil),
	}

	// create the role of a doctor and patient
	_, err = testData.DB.Exec(`insert into role_type (role_type_tag) values ('DOCTOR'),('PATIENT')`)
	if err != nil {
		t.Fatal("Unable to create the provider role of DOCTOR " + err.Error())
	}

	dispatch.Default = dispatch.New()
	environment.SetCurrent("test")
	testData.Config = &router.Config{
		DataAPI: testData.DataApi,
		AuthAPI: testData.AuthApi,
		AddressValidationAPI: &address.StubAddressValidationService{
			CityStateToReturn: &address.CityState{
				City:              "San Francisco",
				State:             "California",
				StateAbbreviation: "CA",
			},
		},
		PaymentAPI: &payment.StubPaymentService{},
		NotifyConfigs: (*config.NotificationConfigs)(&map[string]*config.NotificationConfig{
			"iOS-Patient-Feature": &config.NotificationConfig{
				SNSApplicationEndpoint: "endpoint",
			},
		}),
		NotificationManager: notify.NewManager(testData.DataApi, nil, nil, &email.TestService{}, "", "", nil, metrics.NewRegistry()),
		ERxStatusQueue:      &common.SQSQueue{QueueService: &sqs.StubSQS{}, QueueUrl: "local-erx"},
		ERxAPI:              &erx.StubErxService{SelectedMedicationToReturn: &common.Treatment{}},
		Stores: map[string]storage.Store{
			"photos": storage.NewS3(testData.AWSAuth, "us-east-1", "test-spruce-storage", "photos"),
		},
		SNSClient:       &sns.MockSNS{PushEndpointToReturn: "push_endpoint"},
		MetricsRegistry: metrics.NewRegistry(),
		CloudStorageAPI: testData.CloudStorageService,
		ERxRouting:      true,
		APISubdomain:    "api.spruce.local",
		WebSubdomain:    "www.spruce.local",
		EmailService:    &email.TestService{},
	}

	return testData
}
