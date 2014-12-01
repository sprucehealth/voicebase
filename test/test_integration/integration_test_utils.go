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
	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/pharmacy"
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
		doctorID = GetDoctorIdOfCurrentDoctor(testData, t)
	}

	accountID, err := testData.DataApi.GetAccountIDFromDoctorID(doctorID)
	if err != nil {
		t.Fatalf("Failed to get account ID: %s", err.Error())
	}

	var token string
	err = testData.DB.QueryRow(`SELECT token FROM auth_token WHERE account_id = ?`, accountID).Scan(&token)
	if err == sql.ErrNoRows {
		token, err = testData.AuthApi.CreateToken(accountID, "testclient", true)
		if err != nil {
			t.Fatalf("Failed to create an auth token: %s", err.Error())
		}
	} else if err != nil {
		t.Fatal(err.Error())
	}

	return &apiclient.DoctorClient{
		BaseURL:   testData.APIServer.URL,
		AuthToken: token,
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

func CreateRandomPatientVisitInState(state string, t *testing.T, testData *TestData) *patient.PatientVisitResponse {
	pr := SignupRandomTestPatientInState(state, t, testData)
	pv := CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	AddTestPharmacyForPatient(pr.Patient.PatientId.Int64(), testData, t)
	AddTestAddressForPatient(pr.Patient.PatientId.Int64(), testData, t)

	answerIntakeRequestBody := PrepareAnswersForQuestionsInPatientVisit(pv, t)
	SubmitAnswersIntakeForPatient(pr.Patient.PatientId.Int64(), pr.Patient.AccountId.Int64(),
		answerIntakeRequestBody, testData, t)
	SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(), pv.PatientVisitId, testData, t)
	return pv
}

func CreateRandomAdmin(t *testing.T, testData *TestData) *common.Patient {
	pr := SignupRandomTestPatient(t, testData)
	patient, err := testData.DataApi.GetPatientFromId(pr.Patient.PatientId.Int64())
	test.OK(t, err)
	// update the role to be that of an admin person
	_, err = testData.DB.Exec(`update account set 
		role_type_id = (select id from role_type where role_type_tag='ADMIN') where id = ?`, patient.AccountId.Int64())
	test.OK(t, err)
	return patient
}

func GrantDoctorAccessToPatientCase(t *testing.T, testData *TestData, doctor *common.Doctor, patientCaseId int64) {
	jsonData, err := json.Marshal(&doctor_queue.ClaimPatientCaseRequestData{
		PatientCaseId: encoding.NewObjectId(patientCaseId),
	})

	resp, err := testData.AuthPost(testData.APIServer.URL+router.DoctorCaseClaimURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	test.OK(t, err)
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

func CreateRandomPatientVisitAndPickTP(t *testing.T, testData *TestData, doctor *common.Doctor) (*patient.PatientVisitResponse, *common.TreatmentPlan) {
	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv := CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	answerIntakeRequestBody := PrepareAnswersForQuestionsInPatientVisit(pv, t)
	SubmitAnswersIntakeForPatient(pr.Patient.PatientId.Int64(), pr.Patient.AccountId.Int64(), answerIntakeRequestBody, testData, t)
	SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(), pv.PatientVisitId, testData, t)
	patientCase, err := testData.DataApi.GetPatientCaseFromPatientVisitId(pv.PatientVisitId)
	test.OK(t, err)
	GrantDoctorAccessToPatientCase(t, testData, doctor, patientCase.Id.Int64())
	StartReviewingPatientVisit(pv.PatientVisitId, doctor, testData, t)
	doctorPickTreatmentPlanResponse := PickATreatmentPlanForPatientVisit(pv.PatientVisitId, doctor, nil, testData, t)

	return pv, doctorPickTreatmentPlanResponse.TreatmentPlan
}

func CreateAndSubmitPatientVisitWithSpecifiedAnswers(answers map[int64]*apiservice.AnswerToQuestionItem, testData *TestData, t *testing.T) *patient.PatientVisitResponse {
	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv := CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	answerIntake := PrepareAnswersForQuestionsWithSomeSpecifiedAnswers(pv, answers, t)
	SubmitAnswersIntakeForPatient(pr.Patient.PatientId.Int64(),
		pr.Patient.AccountId.Int64(),
		answerIntake, testData, t)
	SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(),
		pv.PatientVisitId, testData, t)

	return pv
}

func SetupActiveCostForAcne(testData *TestData, t *testing.T) {
	// lets introduce a cost for an acne visit
	var skuId int64
	err := testData.DB.QueryRow(`select id from sku where type = 'acne_visit'`).Scan(&skuId)
	test.OK(t, err)

	res, err := testData.DB.Exec(`insert into item_cost (sku_id, status) values (?,?)`, skuId, api.STATUS_ACTIVE)
	test.OK(t, err)

	itemCostId, err := res.LastInsertId()
	test.OK(t, err)
	_, err = testData.DB.Exec(`insert into line_item (currency, description, amount, item_cost_id) values ('USD','Acne Visit',4000,?)`, itemCostId)
	test.OK(t, err)

}

func SetupTestWithActiveCostAndVisitSubmitted(testData *TestData, t *testing.T) (*common.PatientVisit, *common.SQSQueue, *common.Card) {
	// lets introduce a cost for an acne visit
	SetupActiveCostForAcne(testData, t)

	stubSQSQueue := &common.SQSQueue{
		QueueUrl:     "visit_url",
		QueueService: &sqs.StubSQS{},
	}

	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)

	// now lets go ahead and submit a visit
	pv := CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataApi.GetPatientVisitFromId(pv.PatientVisitId)
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
	test.OK(t, testData.DataApi.AddCardForPatient(patientVisit.PatientId.Int64(), card))
	return patientVisit, stubSQSQueue, card
}

func GenerateAppEvent(action, resource string, resourceId, accountId int64, testData *TestData, t *testing.T) {
	jsonData, err := json.Marshal(&app_event.EventRequestData{
		Resource:   resource,
		ResourceId: resourceId,
		Action:     action,
	})
	test.OK(t, err)

	res, err := testData.AuthPost(testData.APIServer.URL+router.AppEventURLPath, "application/json", bytes.NewReader(jsonData), accountId)
	test.OK(t, err)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}
}

func DetermineQuestionIDForTag(questionTag string, testData *TestData, t *testing.T) int64 {
	questionInfo, err := testData.DataApi.GetQuestionInfo(questionTag, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	return questionInfo.QuestionId
}

func DeterminePotentialAnswerIDForTag(potentialAnswerTag string, testData *TestData, t *testing.T) int64 {
	answerInfos, err := testData.DataApi.GetAnswerInfoForTags([]string{potentialAnswerTag}, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	return answerInfos[0].AnswerId
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

func GetQuestionIdForQuestionTag(questionTag string, testData *TestData, t *testing.T) int64 {
	qi, err := testData.DataApi.GetQuestionInfo(questionTag, api.EN_LANGUAGE_ID)
	test.OK(t, err)

	return qi.QuestionId
}

func JSONPOSTRequest(t *testing.T, path string, v interface{}) *http.Request {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(v); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/", body)
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
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
