package test_integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	patientAPIService "github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/sku"
	"github.com/sprucehealth/backend/test"
)

func SignupRandomTestPatientWithPharmacyAndAddress(t *testing.T, testData *TestData) *patientAPIService.PatientSignedupResponse {
	stubAddressValidationAPI := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	stubAddressValidationAPI.CityStateToReturn = &address.CityState{
		City:              "San Francisco",
		State:             "California",
		StateAbbreviation: "CA",
	}
	pr := signupRandomTestPatient("", t, testData)
	AddTestPharmacyForPatient(pr.Patient.PatientID.Int64(), testData, t)
	AddTestAddressForPatient(pr.Patient.PatientID.Int64(), testData, t)
	return pr
}

func SignupRandomTestPatient(t *testing.T, testData *TestData) *patientAPIService.PatientSignedupResponse {
	stubAddressValidationAPI := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	stubAddressValidationAPI.CityStateToReturn = &address.CityState{
		City:              "San Francisco",
		State:             "California",
		StateAbbreviation: "CA",
	}
	pr := signupRandomTestPatient("", t, testData)
	return pr
}

func SignupTestPatientWithEmail(email string, t *testing.T, testData *TestData) *patientAPIService.PatientSignedupResponse {
	stubAddressValidationAPI := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	stubAddressValidationAPI.CityStateToReturn = &address.CityState{
		City:              "San Francisco",
		State:             "California",
		StateAbbreviation: "CA",
	}
	pr := signupRandomTestPatient(email, t, testData)
	return pr
}

func SignupRandomTestPatientInState(state string, t *testing.T, testData *TestData) *patientAPIService.PatientSignedupResponse {
	stubAddressValidationAPI := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	stubAddressValidationAPI.CityStateToReturn = &address.CityState{City: "TestCity",
		State:             state,
		StateAbbreviation: state,
	}
	pr := signupRandomTestPatient("", t, testData)
	return pr
}

func signupRandomTestPatient(email string, t *testing.T, testData *TestData) *patientAPIService.PatientSignedupResponse {
	requestBody := bytes.NewBufferString("first_name=Test&last_name=Test&email=")

	if email == "" {
		requestBody.WriteString(strconv.FormatInt(rand.Int63(), 10))
		requestBody.WriteString("@example.com")
	} else {
		requestBody.WriteString(email)
	}
	requestBody.WriteString("&password=12345&dob=1987-11-08&zip_code=94115&phone=7348465522&gender=male")
	res, err := testData.AuthPost(testData.APIServer.URL+apipaths.PatientSignupURLPath, "application/x-www-form-urlencoded", requestBody, 0)
	if err != nil {
		t.Fatal("Unable to make post request for registering patient: " + err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}

	signedupPatientResponse := &patientAPIService.PatientSignedupResponse{}
	err = json.Unmarshal(body, signedupPatientResponse)
	if err != nil {
		t.Fatal("Unable to parse response from patient signed up")
	}

	return signedupPatientResponse
}

func GetPatientVisitForPatient(patientID int64, testData *TestData, t *testing.T) *patientAPIService.PatientVisitResponse {
	patientVisit, err := testData.DataAPI.GetLastCreatedPatientVisit(patientID)
	if err != nil {
		t.Fatal(err.Error())
	}

	patientVisitLayout, err := patientAPIService.IntakeLayoutForVisit(testData.Config.DataAPI, testData.Config.Stores["media"],
		testData.Config.AuthTokenExpiration, patientVisit)

	if err != nil {
		t.Fatal(err.Error())
	}
	return &patientAPIService.PatientVisitResponse{Status: patientVisit.Status, PatientVisitID: patientVisit.PatientVisitID.Int64(), ClientLayout: patientVisitLayout}
}

func QueryPatientVisit(patientVisitID, patientAccountID int64, headers map[string]string, testData *TestData, t *testing.T) *patientAPIService.PatientVisitResponse {
	req, err := http.NewRequest("GET", testData.APIServer.URL+apipaths.PatientVisitURLPath+"?patient_visit_id="+strconv.FormatInt(patientVisitID, 10), nil)

	if headers != nil {
		for name, value := range headers {
			req.Header.Set(name, value)
		}
	}

	token, err := testData.AuthAPI.GetToken(patientAccountID)
	test.OK(t, err)
	req.Header.Set("Authorization", "token "+token)

	res, err := http.DefaultClient.Do(req)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// lets parse the response
	pv := &patientAPIService.PatientVisitResponse{}
	err = json.NewDecoder(res.Body).Decode(pv)
	test.OK(t, err)
	test.Equals(t, common.PVStatusOpen, pv.Status)

	return pv
}

func CreatePatientVisitForPatient(patientID int64, testData *TestData, t *testing.T) *patientAPIService.PatientVisitResponse {
	patient, err := testData.DataAPI.GetPatientFromID(patientID)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	// register a patient visit for this patient
	request, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.PatientVisitURLPath, nil)
	test.OK(t, err)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("S-Version", "Patient;Test;0.9.5;0001")
	request.Header.Set("S-OS", "iOS;7.1")

	resp, err := testData.AuthPostWithRequest(request, patient.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the response for the new patient visit call: " + err.Error())
	}

	patientVisitResponse := &patientAPIService.PatientVisitResponse{}
	err = json.Unmarshal(body, patientVisitResponse)
	if err != nil {
		t.Fatal("Unable to unmarshall response body into patient visit response: " + err.Error())
	}

	return patientVisitResponse
}

// randomly answering all top level questions in the patient visit, regardless of the condition under which the questions are presented to the user.
// the goal of this is to get all questions answered so as to render the views for the doctor layout, not to test the sanity of the answers the patient inputs.
func PrepareAnswersForQuestionsInPatientVisit(visitID int64, visitLayout *info_intake.InfoIntakeLayout, t *testing.T) *apiservice.IntakeData {
	return prepareAnswersForVisitIntake(visitID, visitLayout, true, nil, t)
}

func PrepareAnswersForQuestionsInPatientVisitWithoutAlerts(pv *patientAPIService.PatientVisitResponse, t *testing.T) *apiservice.IntakeData {
	return prepareAnswersForVisitIntake(pv.PatientVisitID, pv.ClientLayout, false, nil, t)
}

func PrepareAnswersForQuestionsWithSomeSpecifiedAnswers(visitID int64, visitLayout *info_intake.InfoIntakeLayout,
	specifiedAnswers map[int64]*apiservice.QuestionAnswerItem, t *testing.T) *apiservice.IntakeData {
	return prepareAnswersForVisitIntake(visitID, visitLayout, true, specifiedAnswers, t)
}

func prepareAnswersForVisitIntake(visitID int64, visitLayout *info_intake.InfoIntakeLayout, includeAlerts bool,
	specifiedAnswers map[int64]*apiservice.QuestionAnswerItem, t *testing.T) *apiservice.IntakeData {

	intakeData := apiservice.IntakeData{}
	intakeData.PatientVisitID = visitID
	intakeData.Questions = make([]*apiservice.QuestionAnswerItem, 0)

	for _, section := range visitLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {

				if specifiedAnswers != nil && specifiedAnswers[question.QuestionID] != nil {
					intakeData.Questions = append(intakeData.Questions, specifiedAnswers[question.QuestionID])
					continue
				}

				// skip questions to alert on
				if !includeAlerts && question.ToAlert {
					continue
				}

				switch question.QuestionType {
				case info_intake.QUESTION_TYPE_SINGLE_SELECT:
					intakeData.Questions = append(intakeData.Questions, &apiservice.QuestionAnswerItem{
						QuestionID: question.QuestionID,
						AnswerIntakes: []*apiservice.AnswerItem{&apiservice.AnswerItem{
							PotentialAnswerID: question.PotentialAnswers[0].AnswerID,
						},
						},
					})
				case info_intake.QUESTION_TYPE_MULTIPLE_CHOICE:
					intakeData.Questions = append(intakeData.Questions, &apiservice.QuestionAnswerItem{
						QuestionID: question.QuestionID,
						AnswerIntakes: []*apiservice.AnswerItem{
							&apiservice.AnswerItem{
								PotentialAnswerID: question.PotentialAnswers[0].AnswerID,
							},
							&apiservice.AnswerItem{
								PotentialAnswerID: question.PotentialAnswers[1].AnswerID,
							},
						},
					})
				case info_intake.QUESTION_TYPE_AUTOCOMPLETE:
					intakeData.Questions = append(intakeData.Questions, &apiservice.QuestionAnswerItem{
						QuestionID: question.QuestionID,
						AnswerIntakes: []*apiservice.AnswerItem{
							&apiservice.AnswerItem{
								AnswerText: "autocomplete 1",
							},
						},
					})
				case info_intake.QUESTION_TYPE_FREE_TEXT:
					intakeData.Questions = append(intakeData.Questions, &apiservice.QuestionAnswerItem{
						QuestionID: question.QuestionID,
						AnswerIntakes: []*apiservice.AnswerItem{
							&apiservice.AnswerItem{
								AnswerText: "This is a test answer",
							},
						},
					})
				}
			}
		}
	}
	return &intakeData
}

func SubmitAnswersIntakeForPatient(patientID, patientAccountId int64, intakeData *apiservice.IntakeData, testData *TestData, t *testing.T) {
	jsonData, err := json.Marshal(intakeData)
	if err != nil {
		t.Fatalf("Unable to marshal answer intake body: %s", err)
	}
	resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.PatientVisitIntakeURLPath, "application/json", bytes.NewReader(jsonData), patientAccountId)
	if err != nil {
		t.Fatalf("Unable to successfully make request to submit answer intake: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unable to successfuly make call to submit answer intake. Expected 200 but got %d", resp.StatusCode)
	}
}

func SubmitPatientVisitForPatient(patientID, patientVisitID int64, testData *TestData, t *testing.T) {
	patient, err := testData.DataAPI.GetPatientFromID(patientID)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	buffer := bytes.NewBufferString("patient_visit_id=")
	buffer.WriteString(strconv.FormatInt(patientVisitID, 10))

	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.PatientVisitURLPath, "application/x-www-form-urlencoded", buffer, patient.AccountID.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}

func SubmitPhotoSectionsForQuestionInPatientVisit(accountID int64, requestData *patient_visit.PhotoAnswerIntakeRequestData, testData *TestData, t *testing.T) {
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		t.Fatal(err.Error())
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.PatientVisitPhotoAnswerURLPath, "application/json", bytes.NewReader(jsonData), accountID)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected response code %d for photo intake but got %d", http.StatusOK, resp.StatusCode)
	}
}

type LineItem struct {
	Description string `json:"description"`
	Value       string `json:"value"`
}

type CostResponse struct {
	Total     *LineItem   `json:"total"`
	LineItems []*LineItem `json:"line_items"`
}

func QueryCost(accountID int64, skuType sku.SKU, testData *TestData, t *testing.T) (string, []*LineItem) {
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.PatientCostURLPath+"?item_type="+skuType.String(), accountID)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	var response CostResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	test.OK(t, err)
	return response.Total.Value, response.LineItems
}

func AddCreditCardForPatient(patientID int64, testData *TestData, t *testing.T) {
	err := testData.DataAPI.AddCardForPatient(patientID, &common.Card{
		ThirdPartyID: "thirdparty",
		Fingerprint:  "fingerprint",
		Token:        "token",
		Type:         "Visa",
		IsDefault:    true,
		BillingAddress: &common.Address{
			AddressLine1: "addressLine1",
			City:         "San Francisco",
			State:        "CA",
			ZipCode:      "94115",
		},
	})
	test.OK(t, err)
}

func CreateFollowupVisitForPatient(p *common.Patient, t *testing.T, testData *TestData) error {
	_, err := patientAPIService.CreatePendingFollowup(p, testData.DataAPI, testData.AuthAPI,
		testData.Config.Dispatcher, testData.Config.Stores["media"], testData.Config.AuthTokenExpiration)
	return err
}

func SetupFollowupTest(t *testing.T, testData *TestData) {
	// lets setup a cost for followup
	skuID, err := testData.DataAPI.SKUID(sku.AcneFollowup)

	res, err := testData.DB.Exec(`insert into item_cost (sku_id, status) values (?,?)`, skuID, api.STATUS_ACTIVE)
	test.OK(t, err)
	itemCostID, err := res.LastInsertId()
	test.OK(t, err)
	_, err = testData.DB.Exec(`insert into line_item (currency, description, amount, item_cost_id) values ('USD','Acne Followup',2000,?)`, itemCostID)
	test.OK(t, err)

}
