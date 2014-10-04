package test_integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
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
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/third_party/github.com/BurntSushi/toml"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func init() {
	apiservice.Testing = true
	dispatch.Testing = true
	golog.Default().SetLevel(golog.WARN)
}

type SMS struct {
	From, To, Text string
}

type SMSAPI struct {
	Sent []*SMS
	mu   sync.Mutex
}

func (s *SMSAPI) Send(from, to, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sent = append(s.Sent, &SMS{From: from, To: to, Text: text})
	return nil
}

func (s *SMSAPI) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Sent)
}

type TestData struct {
	T                   *testing.T
	DataApi             api.DataAPI
	AuthApi             api.AuthAPI
	SMSAPI              *SMSAPI
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
	return d.AuthPostWithRequest(req, accountID)
}

func (d *TestData) AuthPostJSON(url string, accountID int64, req, res interface{}) (*http.Response, error) {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpRes, err := d.AuthPostWithRequest(httpReq, accountID)
	if err != nil {
		return httpRes, err
	}
	defer httpRes.Body.Close()
	return httpRes, json.NewDecoder(httpRes.Body).Decode(res)
}

func (d *TestData) AuthPostWithRequest(req *http.Request, accountID int64) (*http.Response, error) {
	if accountID > 0 {
		token, err := d.AuthApi.GetToken(accountID)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "token "+token)
	}
	if req.Header.Get("S-Device-ID") == "" {
		req.Header.Set("S-Device-ID", "TEST")
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

	// BOOSTRAP DATA

	// FIX: We shouldn't have to signup this doctor, but currently
	// tests expect a default doctor to exist. Probably should get rid of this and update
	// tests to instantiate a doctor if one is needed
	SignupRandomTestDoctorInState("CA", t, d)

	// Upload first versions of the intake, review and diagnosis layouts
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	AddFileToMultipartWriter(writer, "intake", "intake-1-0-0.json", IntakeFileLocation, t)
	AddFileToMultipartWriter(writer, "review", "review-1-0-0.json", ReviewFileLocation, t)
	AddFileToMultipartWriter(writer, "diagnose", "diagnose-1-0-0.json", DiagnosisFileLocation, t)

	// specify the app versions and the platform information
	AddFieldToMultipartWriter(writer, "patient_app_version", "0.9.5", t)
	AddFieldToMultipartWriter(writer, "doctor_app_version", "1.2.3", t)
	AddFieldToMultipartWriter(writer, "platform", "iOS", t)

	err := writer.Close()
	test.OK(t, err)

	admin := CreateRandomAdmin(t, d)
	resp, err := d.AuthPost(d.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
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

var testPool = make(chan *TestData, 1)
var errCh chan error

func init() {
	go func() {
		for {
			testData, err := setupTest()
			if err != nil {
				errCh <- err
				return
			}
			testPool <- testData
		}
	}()
}

func setupTest() (*TestData, error) {
	testConf, err := getTestConf()
	if err != nil {
		return nil, err
	}
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
		return nil, err
	}

	dbConfig.DatabaseName = strings.TrimSpace(out.String())
	db, err := connectToDB(dbConfig)
	if err != nil {
		return nil, err
	}
	conf := config.BaseConfig{}
	awsAuth, err := conf.AWSAuth()
	if err != nil {
		return nil, err
	}

	cloudStorageService := api.NewCloudStorageService(awsAuth)

	authTokenExpireDuration := time.Minute * 10
	authApi, err := api.NewAuthAPI(db, authTokenExpireDuration, time.Minute*5, authTokenExpireDuration, time.Minute*5, nullHasher{})
	if err != nil {
		return nil, err
	}
	dataAPI, err := api.NewDataService(db, "api.spruce.local")
	if err != nil {
		return nil, err
	}

	testData := &TestData{
		DataApi:             dataAPI,
		AuthApi:             authApi,
		DBConfig:            dbConfig,
		CloudStorageService: cloudStorageService,
		SMSAPI:              &SMSAPI{},
		DB:                  db,
		AWSAuth:             awsAuth,
		ERxApi: erx.NewDoseSpotService(testConf.DoseSpot.ClinicId, testConf.DoseSpot.UserId,
			testConf.DoseSpot.ClinicKey, testConf.DoseSpot.SOAPEndpoint, testConf.DoseSpot.APIEndpoint, nil),
	}

	// create the role of a doctor and patient
	_, err = testData.DB.Exec(`insert into role_type (role_type_tag) values ('DOCTOR'),('PATIENT')`)
	if err != nil {
		return nil, err
	}

	environment.SetCurrent("test")
	testData.Config = &router.Config{
		DataAPI:             testData.DataApi,
		AuthAPI:             testData.AuthApi,
		Dispatcher:          dispatch.New(),
		AuthTokenExpiration: authTokenExpireDuration,
		AnalyticsLogger:     analytics.DebugLogger{},
		AddressValidationAPI: &address.StubAddressValidationService{
			CityStateToReturn: &address.CityState{
				City:              "San Francisco",
				State:             "California",
				StateAbbreviation: "CA",
			},
		},
		PaymentAPI: &StripeStub{},
		NotifyConfigs: (*config.NotificationConfigs)(&map[string]*config.NotificationConfig{
			"iOS-Patient-Feature": &config.NotificationConfig{
				SNSApplicationEndpoint: "endpoint",
			},
		}),
		NotificationManager: notify.NewManager(testData.DataApi, testData.AuthApi, nil, testData.SMSAPI, &email.TestService{}, "", nil, metrics.NewRegistry()),
		ERxStatusQueue:      &common.SQSQueue{QueueService: &sqs.StubSQS{}, QueueUrl: "local-status-erx"},
		ERxRoutingQueue:     &common.SQSQueue{QueueService: &sqs.StubSQS{}, QueueUrl: "local-routing-erx"},
		ERxAPI:              &erx.StubErxService{SelectedMedicationToReturn: &common.Treatment{}},
		MedicalRecordQueue:  &common.SQSQueue{QueueService: &sqs.StubSQS{}, QueueUrl: "local-medrecord"},
		Stores: map[string]storage.Store{
			"media":          storage.NewS3(testData.AWSAuth, "us-east-1", "test-spruce-storage", "media"),
			"thumbnails":     storage.NewS3(testData.AWSAuth, "us-east-1", "test-spruce-storage", "thumbnails"),
			"medicalrecords": storage.NewTestStore(nil),
		},
		SNSClient:           &sns.MockSNS{PushEndpointToReturn: "push_endpoint"},
		MetricsRegistry:     metrics.NewRegistry(),
		CloudStorageAPI:     testData.CloudStorageService,
		DosespotConfig:      &config.DosespotConfig{},
		ERxRouting:          false,
		APIDomain:           "api.spruce.local",
		WebDomain:           "www.spruce.local",
		EmailService:        &email.TestService{},
		SMSAPI:              testData.SMSAPI,
		TwoFactorExpiration: 60,
	}

	return testData, nil
}

func SetupTest(t *testing.T) *TestData {
	CheckIfRunningLocally(t)
	t.Parallel()

	select {
	case err := <-errCh:
		t.Fatal(err)
	case testData := <-testPool:
		return testData
	}

	return nil
}

func getTestConf() (*TestConf, error) {
	testConf := TestConf{}
	fileContents, err := ioutil.ReadFile(os.Getenv(spruceProjectDirEnv) + "/src/github.com/sprucehealth/backend/test/test.conf")
	if err != nil {
		return nil, err
	}
	_, err = toml.Decode(string(fileContents), &testConf)
	if err != nil {
		return nil, err
	}
	return &testConf, nil

}

func connectToDB(dbConfig *TestDBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.DatabaseName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}
