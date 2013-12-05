package integration

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
)

type AnswerIntakeHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

func getQuestionWithTagAndExpectedType(questionTag, questionType string, t *testing.T, testData TestData) int64 {
	questionId, _, questionType, _, _, err := testData.DataApi.GetQuestionInfo(questionTag, 1)
	if err != nil {
		t.Fatal("Unable to query for question q_reason_visit from database: " + err.Error())
	}

	// need to ensure that the question we are trying to get the information for is a single select
	// question type
	if questionType != questionType {
		t.Fatal("Expected q_reason_visit to be q_type_single_select but it's not.")
	}

	return questionId
}

func getAnswerWithTagAndExpectedType(answerTag, answerType string, questionId int64, testData TestData, t *testing.T) int64 {

	potentialAnswers, err := testData.DataApi.GetAnswerInfo(questionId, 1)
	if err != nil {
		t.Fatal("Unable to get answers for question with id " + strconv.FormatInt(questionId, 10))
	}

	expectedAnswerTag := answerTag
	var potentialAnswerId int64
	var potentialAnswerType string
	for _, potentialAnswer := range potentialAnswers {
		if potentialAnswer.AnswerTag == expectedAnswerTag {
			potentialAnswerId = potentialAnswer.PotentialAnswerId
			potentialAnswerType = potentialAnswer.AnswerType
		}
	}

	if potentialAnswerId == 0 {
		t.Fatal("Unable to find the answer for the question with intended answer tag " + expectedAnswerTag)
	}

	if potentialAnswerType != answerType {
		t.Fatalf("Potential answer found does not have matching type. Expected %s, Found %s ", answerType, potentialAnswerType)
	}

	return potentialAnswerId
}

func submitPatientAnswerForVisit(PatientId int64, testData TestData, patientIntakeRequestData string, t *testing.T) {
	answerIntakeHandler := apiservice.NewAnswerIntakeHandler(testData.DataApi)
	patient, err := testData.DataApi.GetPatientFromId(PatientId)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id when trying to enter patient intake: " + err.Error())
	}

	answerIntakeHandler.AccountIdFromAuthToken(patient.AccountId)
	ts := httptest.NewServer(answerIntakeHandler)
	defer ts.Close()

	client := &http.Client{}
	req, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(patientIntakeRequestData))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}
	CheckSuccessfulStatusCode(resp, "Unable to submit a single select answer for patient", t)
}

func TestSingleSelectIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()

	// signup a random test patient for which to answer questions
	patientSignedUpResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionId := getQuestionWithTagAndExpectedType("q_reason_visit", "q_type_single_select", t, testData)
	potentialAnswerId := getAnswerWithTagAndExpectedType("a_acne", "a_type_multiple_choice", questionId, testData, t)

	// lets go ahead and populate a response for the question
	patientIntakeRequestData := fmt.Sprintf(`{"patient_visit_id": %d, "potential_answers": [{"potential_answer_id": %d } ], "question_id": %d }`, patientVisitResponse.PatientVisitId, potentialAnswerId, questionId)

	// now, lets go ahead and answer the question for the patient
	submitPatientAnswerForVisit(patientSignedUpResponse.PatientId, testData, patientIntakeRequestData, t)

	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionId == questionId {
					if question.PatientAnswers == nil || len(question.PatientAnswers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionId)
					}
					for _, patientAnswer := range question.PatientAnswers {
						if patientAnswer.PotentialAnswerId == potentialAnswerId {
							return
						}
					}
				}
			}
		}
	}

	t.Fatalf("While a patient answer exists for question with id %d, unable to find the expected potential answer with id %d", questionId, potentialAnswerId)
}

func TestMultipleChoiceIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()

	// signup a random test patient for which to answer questions
	patientSignedUpResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionId := getQuestionWithTagAndExpectedType("q_acne_prev_treatment_types", "q_type_multiple_choice", t, testData)
	potentialAnswers, err := testData.DataApi.GetAnswerInfo(questionId, 1)
	if err != nil {
		t.Fatal("Unable to get answers for question with id " + strconv.FormatInt(questionId, 10))
	}

	answerIntakeRequestBody := apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = patientVisitResponse.PatientVisitId
	answerIntakeRequestBody.QuestionId = questionId
	for _, potentialAnswer := range potentialAnswers {
		if potentialAnswer.AnswerTag == "a_otc_prev_treatment_type" || potentialAnswer.AnswerTag == "a_prescription_prev_treatment_type" {
			answerIntakeRequestBody.AnswerIntakes = append(answerIntakeRequestBody.AnswerIntakes, &apiservice.AnswerIntake{PotentialAnswerId: potentialAnswer.PotentialAnswerId})
		}
	}

	requestData, err := json.Marshal(&answerIntakeRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body")
	}
	submitPatientAnswerForVisit(patientSignedUpResponse.PatientId, testData, string(requestData), t)
	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionId == questionId {
					if question.PatientAnswers == nil || len(question.PatientAnswers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionId)
					}
					for _, patientAnswer := range question.PatientAnswers {
						answerNotFound := true
						for _, answerIntake := range answerIntakeRequestBody.AnswerIntakes {
							if answerIntake.PotentialAnswerId == patientAnswer.PotentialAnswerId {
								answerNotFound = false
							}
						}
						if answerNotFound {
							t.Fatal("Expected answer not found in patient answer for patient visit when testing for answering of multiple choice questions.")
						}
					}
				}
			}
		}
	}
}

func TestSingleEntryIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()

	// signup a random test patient for which to answer questions
	patientSignedUpResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionId := getQuestionWithTagAndExpectedType("q_condition_for_diagnosis", "q_type_single_entry", t, testData)
	potentialAnswerId := getAnswerWithTagAndExpectedType("a_condition_entry", "a_type_single_entry", questionId, testData, t)
	answerIntakeRequestBody := apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = patientVisitResponse.PatientVisitId
	answerIntakeRequestBody.QuestionId = questionId
	answerIntakeRequestBody.AnswerIntakes = []*apiservice.AnswerIntake{&apiservice.AnswerIntake{PotentialAnswerId: potentialAnswerId, AnswerText: "testAnswer"}}
	requestData, err := json.Marshal(&answerIntakeRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body")
	}
	submitPatientAnswerForVisit(patientSignedUpResponse.PatientId, testData, string(requestData), t)
	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionId == questionId {
					if question.PatientAnswers == nil || len(question.PatientAnswers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionId)
					}
					for _, patientAnswer := range question.PatientAnswers {
						if patientAnswer.PotentialAnswerId == potentialAnswerId && patientAnswer.AnswerText == "testAnswer" {
							return
						}
					}
				}
			}
		}
	}
	t.Fatalf("While an answer for the expected question exists, unable to find the expected answer with id %d for single entry intake test", potentialAnswerId)
}

func TestFreeTextEntryIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()

	// signup a random test patient for which to answer questions
	patientSignedUpResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionId := getQuestionWithTagAndExpectedType("q_changes_acne_worse", "q_type_free_text", t, testData)
	potentialAnswerId := getAnswerWithTagAndExpectedType("a_changes_acne_worse", "a_type_free_text", questionId, testData, t)
	answerIntakeRequestBody := apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = patientVisitResponse.PatientVisitId
	answerIntakeRequestBody.QuestionId = questionId
	freeTextResponse := "This is a free text response that should be accepted as a response for free text."
	answerIntakeRequestBody.AnswerIntakes = []*apiservice.AnswerIntake{&apiservice.AnswerIntake{PotentialAnswerId: potentialAnswerId, AnswerText: freeTextResponse}}
	requestData, err := json.Marshal(&answerIntakeRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body")
	}
	submitPatientAnswerForVisit(patientSignedUpResponse.PatientId, testData, string(requestData), t)
	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionId == questionId {
					if question.PatientAnswers == nil || len(question.PatientAnswers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionId)
					}
					for _, patientAnswer := range question.PatientAnswers {
						if patientAnswer.PotentialAnswerId == potentialAnswerId && patientAnswer.AnswerText == freeTextResponse {
							return
						}
					}
				}
			}
		}
	}
	t.Fatalf("While an answer for the expected question exists, unable to find the expected answer with id %d for free text intake test", potentialAnswerId)
}

func addSubAnswerToAnswerIntake(answerIntake *apiservice.AnswerIntake, subAnswerQuestionId, subAnswerPotentialAnswerId int64) {
	subQuestionAnswerIntake := &apiservice.SubQuestionAnswerIntake{}
	subQuestionAnswerIntake.QuestionId = subAnswerQuestionId
	subQuestionAnswerIntake.AnswerIntakes = []*apiservice.AnswerIntake{&apiservice.AnswerIntake{PotentialAnswerId: subAnswerPotentialAnswerId}}
	if answerIntake.SubQuestionAnswerIntakes == nil {
		answerIntake.SubQuestionAnswerIntakes = make([]*apiservice.SubQuestionAnswerIntake, 0)
	}
	answerIntake.SubQuestionAnswerIntakes = append(answerIntake.SubQuestionAnswerIntakes, subQuestionAnswerIntake)
}

func TestSubQuestionEntryIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()

	// signup a random test patient for which to answer questions
	patientSignedUpResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionId := getQuestionWithTagAndExpectedType("q_acne_prev_treatment_list", "q_type_compound", t, testData)
	potentialAnswerId := getAnswerWithTagAndExpectedType("a_prev_treatment_list_entry", "a_type_autocomplete_entry", questionId, testData, t)

	// lets go ahead and get the question id for the rest of the three questions that we are trying to answer for this particular entry
	howEffectiveQuestionId := getQuestionWithTagAndExpectedType("q_effective_treatment", "q_type_segmented_control", t, testData)
	howEffectiveAnswerId := getAnswerWithTagAndExpectedType("a_effective_treatment_not_very", "a_type_segmented_control", howEffectiveQuestionId, testData, t)

	usingTreatmentQuestionId := getQuestionWithTagAndExpectedType("q_using_treatment", "q_type_segmented_control", t, testData)
	usingTreatmentAnswerId := getAnswerWithTagAndExpectedType("a_using_treatment_yes", "a_type_segmented_control", usingTreatmentQuestionId, testData, t)

	lengthTreatmentQuestionId := getQuestionWithTagAndExpectedType("q_length_treatment", "q_type_segmented_control", t, testData)
	lengthTreatmentAnswerId := getAnswerWithTagAndExpectedType("a_length_treatment_six_eleven_months", "a_type_segmented_control", lengthTreatmentQuestionId, testData, t)

	// answer the question with three drugs that the patient is using
	proactive := "Proactive"
	benzoylPeroxide := "Benzoyl Peroxide"
	neutrogena := "Neutrogena"

	answerIntakeRequestBody := apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = patientVisitResponse.PatientVisitId
	answerIntakeRequestBody.QuestionId = questionId

	proactiveAnswerIntake := &apiservice.AnswerIntake{}
	proactiveAnswerIntake.PotentialAnswerId = potentialAnswerId
	proactiveAnswerIntake.AnswerText = proactive
	addSubAnswerToAnswerIntake(proactiveAnswerIntake, howEffectiveQuestionId, howEffectiveAnswerId)
	addSubAnswerToAnswerIntake(proactiveAnswerIntake, usingTreatmentQuestionId, usingTreatmentAnswerId)
	addSubAnswerToAnswerIntake(proactiveAnswerIntake, lengthTreatmentQuestionId, lengthTreatmentAnswerId)

	benzoylPeroxideAnswerIntake := &apiservice.AnswerIntake{}
	benzoylPeroxideAnswerIntake.PotentialAnswerId = potentialAnswerId
	benzoylPeroxideAnswerIntake.AnswerText = benzoylPeroxide
	addSubAnswerToAnswerIntake(benzoylPeroxideAnswerIntake, howEffectiveQuestionId, howEffectiveAnswerId)
	addSubAnswerToAnswerIntake(benzoylPeroxideAnswerIntake, usingTreatmentQuestionId, usingTreatmentAnswerId)
	addSubAnswerToAnswerIntake(benzoylPeroxideAnswerIntake, lengthTreatmentQuestionId, lengthTreatmentAnswerId)

	neutrogenaAnswerIntake := &apiservice.AnswerIntake{}
	neutrogenaAnswerIntake.PotentialAnswerId = potentialAnswerId
	neutrogenaAnswerIntake.AnswerText = neutrogena
	addSubAnswerToAnswerIntake(neutrogenaAnswerIntake, howEffectiveQuestionId, howEffectiveAnswerId)
	addSubAnswerToAnswerIntake(neutrogenaAnswerIntake, usingTreatmentQuestionId, usingTreatmentAnswerId)
	addSubAnswerToAnswerIntake(neutrogenaAnswerIntake, lengthTreatmentQuestionId, lengthTreatmentAnswerId)

	answerIntakeRequestBody.AnswerIntakes = []*apiservice.AnswerIntake{proactiveAnswerIntake, benzoylPeroxideAnswerIntake, neutrogenaAnswerIntake}

	requestData, err := json.Marshal(&answerIntakeRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body")
	}
	submitPatientAnswerForVisit(patientSignedUpResponse.PatientId, testData, string(requestData), t)
	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionId == questionId {
					if question.PatientAnswers == nil || len(question.PatientAnswers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionId)
					}
					for _, patientAnswer := range question.PatientAnswers {

						if !(patientAnswer.PotentialAnswerId == potentialAnswerId &&
							(patientAnswer.AnswerText == neutrogena || patientAnswer.AnswerText == benzoylPeroxide ||
								patientAnswer.AnswerText == proactive)) {
							t.Fatal("Top level patient answers is not one of the expected answers")
						}
						for _, subAnswer := range patientAnswer.SubAnswers {
							if !(subAnswer.PotentialAnswerId == howEffectiveAnswerId ||
								subAnswer.PotentialAnswerId == usingTreatmentAnswerId ||
								subAnswer.PotentialAnswerId == lengthTreatmentAnswerId) &&
								(subAnswer.QuestionId == howEffectiveQuestionId ||
									subAnswer.QuestionId == usingTreatmentQuestionId ||
									subAnswer.QuestionId == lengthTreatmentQuestionId) {
								t.Fatal("Sub answers to top level answers is not one of the expected answers")
							}

							if subAnswer.AnswerSummary == "" {
								t.Fatalf("The %s potential answer id should have an answer summary", subAnswer.PotentialAnswerId)
							}
						}
					}
				}
			}
		}
	}

	// now update the answer to this question to ensure that we can update answers no problem
	proactiveAnswerIntake.SubQuestionAnswerIntakes = nil
	benzoylPeroxideAnswerIntake.SubQuestionAnswerIntakes = nil
	neutrogenaAnswerIntake.SubQuestionAnswerIntakes = nil
	requestData, err = json.Marshal(&answerIntakeRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body second time around")
	}
	submitPatientAnswerForVisit(patientSignedUpResponse.PatientId, testData, string(requestData), t)
	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)
	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionId == questionId {
					if question.PatientAnswers == nil || len(question.PatientAnswers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionId)
					}
					for _, patientAnswer := range question.PatientAnswers {

						if !(patientAnswer.PotentialAnswerId == potentialAnswerId &&
							(patientAnswer.AnswerText == neutrogena || patientAnswer.AnswerText == benzoylPeroxide ||
								patientAnswer.AnswerText == proactive)) {
							t.Fatal("Top level patient answers is not one of the expected answers")
						}

						if !(patientAnswer.SubAnswers == nil || len(patientAnswer.SubAnswers) == 0) {
							t.Fatal("Subanswers not expected but they still exist in the patient answers")
						}
					}
				}
			}
		}
	}
}

func TestPhotoAnswerIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()

	// signup a random test patient for which to answer questions
	patientSignedUpResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)
	questionId := getQuestionWithTagAndExpectedType("q_chest_photo_intake", "q_type_single_photo", t, testData)
	potentialAnswerId := getAnswerWithTagAndExpectedType("a_chest_phota_intake", "a_type_photo_entry_chest", questionId, testData, t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	// uploading any file as a photo for now
	part, err := writer.CreateFormFile("photo", "example.jpg")
	if err != nil {
		t.Fatal("Unable to create a form file with a sample file")
	}

	file, err := os.Open("../info_intake/condition_intake.json")
	if err != nil {
		t.Fatal("Unable to open file for uploading: " + err.Error())
	}
	_, err = io.Copy(part, file)
	if err != nil {
		t.Fatal("Unable to copy contents of file into multipart form data: " + err.Error())
	}

	writer.WriteField("question_id", strconv.FormatInt(questionId, 10))
	writer.WriteField("potential_answer_id", strconv.FormatInt(potentialAnswerId, 10))
	writer.WriteField("patient_visit_id", strconv.FormatInt(patientVisitResponse.PatientVisitId, 10))

	err = writer.Close()
	if err != nil {
		t.Fatal("Unable to create multi-form data. Error when trying to close writer: " + err.Error())
	}

	photoAnswerIntakeHandler := apiservice.NewPhotoAnswerIntakeHandler(testData.DataApi, testData.CloudStorageService, "cases-bucket-integ", "us-east-1", 1*1024*1024)
	patient, err := testData.DataApi.GetPatientFromId(patientSignedUpResponse.PatientId)
	if err != nil {
		t.Fatal("Unable to retrieve patient data given the patient id: " + err.Error())
	}
	photoAnswerIntakeHandler.AccountIdFromAuthToken(patient.AccountId)
	ts := httptest.NewServer(photoAnswerIntakeHandler)
	defer ts.Close()

	client := &http.Client{}
	req, _ := http.NewRequest("POST", ts.URL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Unable to submit photo answer for patient: " + err.Error())
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read the body of the response when trying to submit photo answer for patient: " + err.Error())
	}
	CheckSuccessfulStatusCode(resp, "Unable to submit photo answer for patient: "+string(responseBody), t)

	// get the patient visit again to get the patient answer in there
	patientVisitResponse = GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionId == questionId {
					if question.PatientAnswers == nil || len(question.PatientAnswers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionId)
					}
					for _, patientAnswer := range question.PatientAnswers {
						if patientAnswer.PotentialAnswerId == potentialAnswerId &&
							patientAnswer.ObjectUrl != "" {
							buffer := bytes.NewBufferString(strconv.FormatInt(patientVisitResponse.PatientVisitId, 10))
							buffer.WriteString("/")
							buffer.WriteString(strconv.FormatInt(patientAnswer.PatientAnswerId, 10))
							buffer.WriteString(".jpg")
							err = testData.CloudStorageService.DeleteObjectAtLocation("cases-bucket-integ", buffer.String(), "us-east-1")
							if err != nil {
								t.Fatalf("Unable to delete object at location %s : %s ", patientAnswer.ObjectUrl, err.Error())
							}
							return
						}
					}
				}
			}
		}
	}
	t.Fatal("Photo answer submitted not found as patient answer")
}
