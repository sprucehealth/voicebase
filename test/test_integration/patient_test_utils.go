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
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/info_intake"
	patientApiService "github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/test"
)

func SignupRandomTestPatient(t *testing.T, testData *TestData) *patientApiService.PatientSignedupResponse {
	stubAddressValidationAPI := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	stubAddressValidationAPI.CityStateToReturn = &address.CityState{
		City:              "San Francisco",
		State:             "California",
		StateAbbreviation: "CA",
	}
	return signupRandomTestPatient(t, testData)
}

func signupRandomTestPatient(t *testing.T, testData *TestData) *patientApiService.PatientSignedupResponse {
	requestBody := bytes.NewBufferString("first_name=Test&last_name=Test&email=")
	requestBody.WriteString(strconv.FormatInt(rand.Int63(), 10))
	requestBody.WriteString("@example.com&password=12345&dob=1987-11-08&zip_code=94115&phone=7348465522&gender=male")
	res, err := testData.AuthPost(testData.APIServer.URL+router.PatientSignupURLPath, "application/x-www-form-urlencoded", requestBody, 0)
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

	signedupPatientResponse := &patientApiService.PatientSignedupResponse{}
	err = json.Unmarshal(body, signedupPatientResponse)
	if err != nil {
		t.Fatal("Unable to parse response from patient signed up")
	}

	AddTestPharmacyForPatient(signedupPatientResponse.Patient.PatientId.Int64(), testData, t)
	AddTestAddressForPatient(signedupPatientResponse.Patient.PatientId.Int64(), testData, t)

	return signedupPatientResponse
}

func SignupRandomTestPatientInState(state string, t *testing.T, testData *TestData) *patientApiService.PatientSignedupResponse {
	stubAddressValidationAPI := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	stubAddressValidationAPI.CityStateToReturn = &address.CityState{City: "TestCity",
		State:             state,
		StateAbbreviation: state,
	}
	return signupRandomTestPatient(t, testData)
}

func GetPatientVisitForPatient(patientId int64, testData *TestData, t *testing.T) *patientApiService.PatientVisitResponse {
	patientVisitId, err := testData.DataApi.GetLastCreatedPatientVisitIdForPatient(patientId)
	if err != nil {
		t.Fatal(err.Error())
	}

	patientVisit, err := testData.DataApi.GetPatientVisitFromId(patientVisitId)
	if err != nil {
		t.Fatal(err.Error())
	}

	r, err := http.NewRequest("GET", "localhost", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	patientVisitLayout, err := patientApiService.GetPatientVisitLayout(testData.Config.DataAPI, testData.Config.Stores["media"],
		testData.Config.AuthTokenExpiration, patientVisit, r)

	if err != nil {
		t.Fatal(err.Error())
	}
	return &patientApiService.PatientVisitResponse{Status: patientVisit.Status, PatientVisitId: patientVisitId, ClientLayout: patientVisitLayout}
}

func CreatePatientVisitForPatient(patientId int64, testData *TestData, t *testing.T) *patientApiService.PatientVisitResponse {
	patient, err := testData.DataApi.GetPatientFromId(patientId)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	// register a patient visit for this patient
	request, err := http.NewRequest("POST", testData.APIServer.URL+router.PatientVisitURLPath, nil)
	test.OK(t, err)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("S-Version", "Patient;Test;0.9.5;0001")
	request.Header.Set("S-OS", "iOS;7.1")

	resp, err := testData.AuthPostWithRequest(request, patient.AccountId.Int64())
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

	patientVisitResponse := &patientApiService.PatientVisitResponse{}
	err = json.Unmarshal(body, patientVisitResponse)
	if err != nil {
		t.Fatal("Unable to unmarshall response body into patient visit response: " + err.Error())
	}

	return patientVisitResponse
}

// randomly answering all top level questions in the patient visit, regardless of the condition under which the questions are presented to the user.
// the goal of this is to get all questions answered so as to render the views for the doctor layout, not to test the sanity of the answers the patient inputs.
func PrepareAnswersForQuestionsInPatientVisit(patientVisitResponse *patientApiService.PatientVisitResponse, t *testing.T) *apiservice.AnswerIntakeRequestBody {
	return prepareAnswersForVisitIntake(patientVisitResponse, true, t)
}

func PrepareAnswersForQuestionsInPatientVisitWithoutAlerts(patientVisitResponse *patientApiService.PatientVisitResponse, t *testing.T) *apiservice.AnswerIntakeRequestBody {
	return prepareAnswersForVisitIntake(patientVisitResponse, false, t)
}

func prepareAnswersForVisitIntake(patientVisitResponse *patientApiService.PatientVisitResponse, includeAlerts bool, t *testing.T) *apiservice.AnswerIntakeRequestBody {
	answerIntakeRequestBody := apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = patientVisitResponse.PatientVisitId
	answerIntakeRequestBody.Questions = make([]*apiservice.AnswerToQuestionItem, 0)
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {

				// skip questions to alert on
				if !includeAlerts && question.ToAlert {
					continue
				}

				switch question.QuestionType {
				case info_intake.QUESTION_TYPE_SINGLE_SELECT:
					answerIntakeRequestBody.Questions = append(answerIntakeRequestBody.Questions, &apiservice.AnswerToQuestionItem{
						QuestionId: question.QuestionId,
						AnswerIntakes: []*apiservice.AnswerItem{&apiservice.AnswerItem{
							PotentialAnswerId: question.PotentialAnswers[0].AnswerId,
						},
						},
					})
				case info_intake.QUESTION_TYPE_MULTIPLE_CHOICE:
					answerIntakeRequestBody.Questions = append(answerIntakeRequestBody.Questions, &apiservice.AnswerToQuestionItem{
						QuestionId: question.QuestionId,
						AnswerIntakes: []*apiservice.AnswerItem{
							&apiservice.AnswerItem{
								PotentialAnswerId: question.PotentialAnswers[0].AnswerId,
							},
							&apiservice.AnswerItem{
								PotentialAnswerId: question.PotentialAnswers[1].AnswerId,
							},
						},
					})
				case info_intake.QUESTION_TYPE_AUTOCOMPLETE:
					answerIntakeRequestBody.Questions = append(answerIntakeRequestBody.Questions, &apiservice.AnswerToQuestionItem{
						QuestionId: question.QuestionId,
						AnswerIntakes: []*apiservice.AnswerItem{
							&apiservice.AnswerItem{
								AnswerText: "autocomplete 1",
							},
						},
					})
				case info_intake.QUESTION_TYPE_FREE_TEXT:
					answerIntakeRequestBody.Questions = append(answerIntakeRequestBody.Questions, &apiservice.AnswerToQuestionItem{
						QuestionId: question.QuestionId,
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
	return &answerIntakeRequestBody
}

func SubmitAnswersIntakeForPatient(patientId, patientAccountId int64, answerIntakeRequestBody *apiservice.AnswerIntakeRequestBody, testData *TestData, t *testing.T) {
	jsonData, err := json.Marshal(answerIntakeRequestBody)
	if err != nil {
		t.Fatalf("Unable to marshal answer intake body: %s", err)
	}
	resp, err := testData.AuthPost(testData.APIServer.URL+router.PatientVisitIntakeURLPath, "application/json", bytes.NewReader(jsonData), patientAccountId)
	if err != nil {
		t.Fatalf("Unable to successfully make request to submit answer intake: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unable to successfuly make call to submit answer intake. Expected 200 but got %d", resp.StatusCode)
	}
}

func SubmitPatientVisitForPatient(patientId, patientVisitId int64, testData *TestData, t *testing.T) {
	patient, err := testData.DataApi.GetPatientFromId(patientId)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	buffer := bytes.NewBufferString("patient_visit_id=")
	buffer.WriteString(strconv.FormatInt(patientVisitId, 10))

	resp, err := testData.AuthPut(testData.APIServer.URL+router.PatientVisitURLPath, "application/x-www-form-urlencoded", buffer, patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}
	defer resp.Body.Close()

	test.Equals(t, http.StatusOK, resp.StatusCode)
}

func SubmitPhotoSectionsForQuestionInPatientVisit(accountId int64, requestData *patient_visit.PhotoAnswerIntakeRequestData, testData *TestData, t *testing.T) {
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		t.Fatal(err.Error())
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+router.PatientVisitPhotoAnswerURLPath, "application/json", bytes.NewReader(jsonData), accountId)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected response code %d for photo intake but got %d", http.StatusOK, resp.StatusCode)
	}
}
