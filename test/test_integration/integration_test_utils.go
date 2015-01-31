package test_integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"runtime"
	"testing"

	_ "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiclient"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
)

var (
	CannotRunTestLocally       = errors.New("test: The test database is not set. Skipping test")
	spruceProjectDirEnv        = "GOPATH"
	IntakeFileLocation         = "../../info_intake/major-intake-test.json"
	ReviewFileLocation         = "../../info_intake/major-review-test.json"
	FollowupIntakeFileLocation = "../../info_intake/major-followup-intake-test.json"
	FollowupReviewFileLocation = "../../info_intake/major-followup-review-test.json"
	DiagnosisFileLocation      = "../../info_intake/diagnose-1-0-0.json"
	SKUAcneFollowup            = "acne_followup"
	SKUAcneVisit               = "acne_visit"
)

type TestDosespotConfig struct {
	ClinicID     int64  `long:"clinic_id" description:"Clinic Id for dosespot"`
	ClinicKey    string `long:"clinic_key" description:"Clinic Key for dosespot"`
	UserID       int64  `long:"user_id" description:"User Id for dosespot"`
	SOAPEndpoint string `long:"soap_endpoint" description:"SOAP endpoint"`
	APIEndpoint  string `long:"api_endpoint" description:"API endpoint where soap actions are defined"`
}

type TestConf struct {
	DB       config.DB          `group:"Database" toml:"database"`
	DoseSpot TestDosespotConfig `group:"Dosespot" toml:"dosespot"`
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

func CheckIfRunningLocally(t *testing.T) {
	// if the TEST_DB is not set in the environment, we assume
	// that we are running these tests locally, in which case
	// we exit the tests with a warning
	if os.Getenv(spruceProjectDirEnv) == "" {
		t.Skip("WARNING: The test database is not set. Skipping test ")
	}
}

func DoctorClient(testData *TestData, t *testing.T, doctorID int64) *apiclient.DoctorClient {
	if doctorID == 0 {
		doctorID = GetDoctorIDOfCurrentDoctor(testData, t)
	}

	accountID, err := testData.DataAPI.GetAccountIDFromDoctorID(doctorID)
	if err != nil {
		t.Fatalf("Failed to get account ID: %s", err.Error())
	}

	var token string
	err = testData.DB.QueryRow(`SELECT token FROM auth_token WHERE account_id = ?`, accountID).Scan(&token)
	if err == sql.ErrNoRows {
		token, err = testData.AuthAPI.CreateToken(accountID, "testclient", true)
		if err != nil {
			t.Fatalf("Failed to create an auth token: %s", err.Error())
		}
	} else if err != nil {
		t.Fatal(err.Error())
	}

	return &apiclient.DoctorClient{
		Config: apiclient.Config{
			BaseURL:   testData.APIServer.URL,
			AuthToken: token,
		},
	}
}

func PatientClient(testData *TestData, t *testing.T, patientID int64) *apiclient.PatientClient {
	var token string

	if patientID != 0 {
		patient, err := testData.DataAPI.GetPatientFromID(patientID)
		if err != nil {
			t.Fatalf("Failed to get patient: %s", err.Error())
		}
		accountID := patient.AccountID.Int64()

		err = testData.DB.QueryRow(`SELECT token FROM auth_token WHERE account_id = ?`, accountID).Scan(&token)
		if err == sql.ErrNoRows {
			token, err = testData.AuthAPI.CreateToken(accountID, "testclient", true)
			if err != nil {
				t.Fatalf("Failed to create an auth token: %s", err.Error())
			}
		} else if err != nil {
			t.Fatal(err.Error())
		}
	}

	return &apiclient.PatientClient{
		Config: apiclient.Config{
			BaseURL:   testData.APIServer.URL,
			AuthToken: token,
		},
	}
}

func GetDoctorIDOfCurrentDoctor(testData *TestData, t *testing.T) int64 {
	// get the current primary doctor
	var doctorID int64
	err := testData.DB.QueryRow(`select provider_id from care_provider_state_elligibility
							inner join role_type on role_type_id = role_type.id
							inner join care_providing_state on care_providing_state_id = care_providing_state.id
							where role_type_tag='DOCTOR' and care_providing_state.state = 'CA'`).Scan(&doctorID)
	if err != nil {
		t.Fatal("Unable to query for doctor that is elligible to diagnose in CA: " + err.Error())
	}
	return doctorID
}

func CreateRandomPatientVisitInState(state string, t *testing.T, testData *TestData) *patient.PatientVisitResponse {
	pr := SignupRandomTestPatientInState(state, t, testData)
	pv := CreatePatientVisitForPatient(pr.Patient.PatientID.Int64(), testData, t)
	AddTestPharmacyForPatient(pr.Patient.PatientID.Int64(), testData, t)
	AddTestAddressForPatient(pr.Patient.PatientID.Int64(), testData, t)

	intakeData := PrepareAnswersForQuestionsInPatientVisit(pv.PatientVisitID, pv.ClientLayout, t)
	SubmitAnswersIntakeForPatient(pr.Patient.PatientID.Int64(), pr.Patient.AccountID.Int64(),
		intakeData, testData, t)
	SubmitPatientVisitForPatient(pr.Patient.PatientID.Int64(), pv.PatientVisitID, testData, t)
	return pv
}

func CreateRandomAdmin(t *testing.T, testData *TestData) *common.Patient {
	pr := SignupRandomTestPatient(t, testData)
	patient, err := testData.DataAPI.GetPatientFromID(pr.Patient.PatientID.Int64())
	test.OK(t, err)
	// update the role to be that of an admin person
	_, err = testData.DB.Exec(`update account set 
		role_type_id = (select id from role_type where role_type_tag='ADMIN') where id = ?`, patient.AccountID.Int64())
	test.OK(t, err)
	return patient
}

func GrantDoctorAccessToPatientCase(t *testing.T, testData *TestData, doctor *common.Doctor, patientCaseID int64) {
	jsonData, err := json.Marshal(&doctor_queue.ClaimPatientCaseRequestData{
		PatientCaseID: encoding.NewObjectID(patientCaseID),
	})

	resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.DoctorCaseClaimURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountID.Int64())
	test.OK(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected response %d instead got %d", http.StatusOK, resp.StatusCode)
	}
}

func AddTestAddressForPatient(patientID int64, testData *TestData, t *testing.T) {
	if err := testData.DataAPI.UpdateDefaultAddressForPatient(patientID, &common.Address{
		AddressLine1: "123 Street",
		AddressLine2: "Apt 123",
		City:         "San Francisco",
		State:        "CA",
		ZipCode:      "94115",
	}); err != nil {
		t.Fatal(err)
	}
}

func AddTestPharmacyForPatient(patientID int64, testData *TestData, t *testing.T) {
	if err := testData.DataAPI.UpdatePatientPharmacy(patientID, &pharmacy.PharmacyData{
		SourceID:     123,
		PatientID:    patientID,
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

func CreateTestResourceGuides(t *testing.T, testData *TestData) (int64, []int64) {
	secID, err := testData.DataAPI.CreateResourceGuideSection(&common.ResourceGuideSection{
		Ordinal: 1,
		Title:   "Test Section",
	})
	test.OK(t, err)

	guide1ID, err := testData.DataAPI.CreateResourceGuide(&common.ResourceGuide{
		SectionID: secID,
		Ordinal:   1,
		Title:     "Guide 1",
		PhotoURL:  "http://example.com/blah.png",
		Layout:    &struct{}{},
	})
	test.OK(t, err)

	guide2ID, err := testData.DataAPI.CreateResourceGuide(&common.ResourceGuide{
		SectionID: secID,
		Ordinal:   2,
		Title:     "Guide 2",
		PhotoURL:  "http://example.com/blah.png",
		Layout:    &struct{}{},
	})
	test.OK(t, err)

	return secID, []int64{guide1ID, guide2ID}
}

func CreateRandomPatientVisitAndPickTP(t *testing.T, testData *TestData, doctor *common.Doctor) (*patient.PatientVisitResponse, *common.TreatmentPlan) {
	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	return CreatePatientVisitAndPickTP(t, testData, pr.Patient, doctor)
}

func CreatePatientVisitAndPickTP(t *testing.T, testData *TestData, patient *common.Patient, doctor *common.Doctor) (*patient.PatientVisitResponse, *common.TreatmentPlan) {
	pv := CreatePatientVisitForPatient(patient.PatientID.Int64(), testData, t)
	intakeData := PrepareAnswersForQuestionsInPatientVisit(pv.PatientVisitID, pv.ClientLayout, t)
	SubmitAnswersIntakeForPatient(patient.PatientID.Int64(), patient.AccountID.Int64(), intakeData, testData, t)
	SubmitPatientVisitForPatient(patient.PatientID.Int64(), pv.PatientVisitID, testData, t)
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	GrantDoctorAccessToPatientCase(t, testData, doctor, patientCase.ID.Int64())
	StartReviewingPatientVisit(pv.PatientVisitID, doctor, testData, t)
	doctorPickTreatmentPlanResponse := PickATreatmentPlanForPatientVisit(pv.PatientVisitID, doctor, nil, testData, t)
	role := api.DOCTOR_ROLE
	if doctor.IsMA {
		role = api.MA_ROLE
	}
	tp, err := responses.TransformTPFromResponse(testData.DataAPI, doctorPickTreatmentPlanResponse.TreatmentPlan, doctor.DoctorID.Int64(), role)
	if err != nil {
		t.Fatal(err)
	}
	return pv, tp
}

func CreateAndSubmitPatientVisitWithSpecifiedAnswers(answers map[int64]*apiservice.QuestionAnswerItem, testData *TestData, t *testing.T) *patient.PatientVisitResponse {
	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv := CreatePatientVisitForPatient(pr.Patient.PatientID.Int64(), testData, t)
	answerIntake := PrepareAnswersForQuestionsWithSomeSpecifiedAnswers(pv.PatientVisitID, pv.ClientLayout, answers, t)
	SubmitAnswersIntakeForPatient(pr.Patient.PatientID.Int64(),
		pr.Patient.AccountID.Int64(),
		answerIntake, testData, t)
	SubmitPatientVisitForPatient(pr.Patient.PatientID.Int64(),
		pv.PatientVisitID, testData, t)

	return pv
}

func SetupActiveCostForAcne(testData *TestData, t *testing.T) {
	// lets introduce a cost for an acne visit
	var skuID int64
	err := testData.DB.QueryRow(`select id from sku where type = 'acne_visit'`).Scan(&skuID)
	test.OK(t, err)

	res, err := testData.DB.Exec(`insert into item_cost (sku_id, status) values (?,?)`, skuID, api.STATUS_ACTIVE)
	test.OK(t, err)

	itemCostID, err := res.LastInsertId()
	test.OK(t, err)
	_, err = testData.DB.Exec(`insert into line_item (currency, description, amount, item_cost_id) values ('USD','Acne Visit',4000,?)`, itemCostID)
	test.OK(t, err)

}

func SetupTestWithActiveCostAndVisitSubmitted(testData *TestData, t *testing.T) (*common.PatientVisit, *common.SQSQueue, *common.Card) {
	// lets introduce a cost for an acne visit
	SetupActiveCostForAcne(testData, t)

	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &sqs.StubSQS{},
	}

	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)

	// now lets go ahead and submit a visit
	pv := CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pv.PatientVisitID)
	test.OK(t, err)

	// lets go ahead and add a default card for the patient
	card := &common.Card{
		ThirdPartyID: "thirdparty",
		Fingerprint:  "fingerprint",
		Token:        "token",
		Type:         "Visa",
		BillingAddress: &common.Address{
			AddressLine1: "addressLine1",
			City:         "San Francisco",
			State:        "CA",
			ZipCode:      "94115",
		},
		IsDefault: true,
	}
	test.OK(t, testData.DataAPI.AddCardForPatient(patientVisit.PatientID.Int64(), card))
	return patientVisit, stubSQSQueue, card
}

func GenerateAppEvent(action, resource string, resourceID, accountID int64, testData *TestData, t *testing.T) {
	jsonData, err := json.Marshal(&app_event.EventRequestData{
		Resource:   resource,
		ResourceID: resourceID,
		Action:     action,
	})
	test.OK(t, err)

	res, err := testData.AuthPost(testData.APIServer.URL+apipaths.AppEventURLPath, "application/json", bytes.NewReader(jsonData), accountID)
	test.OK(t, err)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}
}

func DetermineQuestionIDForTag(questionTag string, version int64, testData *TestData, t *testing.T) int64 {
	questionInfo, err := testData.DataAPI.GetQuestionInfo(questionTag, api.EN_LANGUAGE_ID, version)
	test.OK(t, err)
	return questionInfo.QuestionID
}

func DeterminePotentialAnswerIDForTag(potentialAnswerTag string, testData *TestData, t *testing.T) int64 {
	answerInfos, err := testData.DataAPI.GetAnswerInfoForTags([]string{potentialAnswerTag}, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	return answerInfos[0].AnswerID
}

func AddFieldToMultipartWriter(writer *multipart.Writer, fieldName, fieldValue string, t *testing.T) {
	field, err := writer.CreateFormField(fieldName)
	test.OK(t, err)
	_, err = field.Write([]byte(fieldValue))
	test.OK(t, err)
}

func AddFileToMultipartWriter(writer *multipart.Writer, layoutType string, fileName, fileLocation string, t *testing.T) {
	part, err := writer.CreateFormFile(layoutType, fileName)
	test.OK(t, err)
	data, err := ioutil.ReadFile(fileLocation)
	test.OK(t, err)
	_, err = part.Write(data)
	test.OK(t, err)
}

type UploadLayoutConfig struct {
	IntakeFileName     string
	IntakeFileLocation string
	ReviewFileName     string
	ReviewFileLocation string
	PatientAppVersion  string
	DoctorAppVersion   string
	Platform           string
}

func UploadIntakeLayoutConfiguration(config *UploadLayoutConfig, testData *TestData, t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if config.IntakeFileName != "" && config.IntakeFileLocation != "" {
		AddFileToMultipartWriter(writer,
			"intake",
			config.IntakeFileName,
			config.IntakeFileLocation, t)
	}
	if config.ReviewFileName != "" && config.ReviewFileLocation != "" {
		AddFileToMultipartWriter(writer,
			"review",
			config.ReviewFileName,
			config.ReviewFileLocation, t)
	}

	// specify the app versions and the platform information
	AddFieldToMultipartWriter(writer, "patient_app_version", config.PatientAppVersion, t)
	AddFieldToMultipartWriter(writer, "doctor_app_version", config.DoctorAppVersion, t)
	AddFieldToMultipartWriter(writer, "platform", config.Platform, t)

	err := writer.Close()
	test.OK(t, err)

	resp, err := testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}

func UploadDetailsLayoutForDiagnosis(layout string, accountID int64, testData *TestData, t *testing.T) {
	var b bytes.Buffer
	_, err := b.WriteString(layout)
	test.OK(t, err)
	res, err := testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout/diagnosis`, "application/json", &b, testData.AdminUser)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
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

func GetQuestionIDForQuestionTag(questionTag string, version int64, testData *TestData, t *testing.T) int64 {
	qi, err := testData.DataAPI.GetQuestionInfo(questionTag, api.EN_LANGUAGE_ID, version)
	test.OK(t, err)

	return qi.QuestionID
}

func CallerString(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
	short := file
	depth := 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			depth++
			if depth == 2 {
				break
			}
		}
	}
	return fmt.Sprintf("%s:%d", short, line)
}
