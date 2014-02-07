package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/erx"
	thriftapi "carefront/thrift/api"
)

func SignupRandomTestDoctor(t *testing.T, dataApi api.DataAPI, authApi thriftapi.Auth) (signedupDoctorResponse *apiservice.DoctorSignedupResponse, email, password string) {
	authHandler := &apiservice.SignupDoctorHandler{AuthApi: authApi, DataApi: dataApi}
	ts := httptest.NewServer(authHandler)
	defer ts.Close()

	requestBody := bytes.NewBufferString("first_name=Test&last_name=Test&email=")
	email = strconv.FormatInt(time.Now().Unix(), 10) + "@example.com"
	password = "12345"
	requestBody.WriteString(email)
	requestBody.WriteString("&password=")
	requestBody.WriteString(password)
	requestBody.WriteString("&dob=11/08/1987&gender=male")
	res, err := authPost(ts.URL, "application/x-www-form-urlencoded", requestBody, 0)
	if err != nil {
		t.Fatal("Unable to make post request for registering patient: " + err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}
	CheckSuccessfulStatusCode(res, fmt.Sprintf("Unable to make success request to signup patient. Here's the code returned %d and here's the body of the request %s", res.StatusCode, body), t)

	signedupDoctorResponse = &apiservice.DoctorSignedupResponse{}
	err = json.Unmarshal(body, signedupDoctorResponse)
	if err != nil {
		t.Fatal("Unable to parse response from patient signed up")
	}
	return signedupDoctorResponse, email, password
}

func setupErxAPI(t *testing.T) *erx.DoseSpotService {
	clinicKey := os.Getenv("DOSESPOT_CLINIC_KEY")
	userId := os.Getenv("DOSESPOT_USER_ID")
	clinicId := os.Getenv("DOSESPOT_CLINIC_ID")

	if clinicKey == "" {
		t.Log("WARNING: skipping doctor drug search test since the dosespot ids are not present as environment variables")
		t.SkipNow()
	}

	erx := erx.NewDoseSpotService(clinicId, clinicKey, userId, nil)
	return erx
}

func submitPatientVisitDiagnosis(PatientVisitId int64, doctor *common.Doctor, testData TestData, t *testing.T) (diagnosisQuestionId, severityQuestionId, acneTypeQuestionId int64) {
	answerIntakeRequestBody := &apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = PatientVisitId

	if qi, err := testData.DataApi.GetQuestionInfo("q_acne_diagnosis", 1); err != nil {
		t.Fatalf("Unable to get the questionIds for the question tags requested for the doctor to diagnose patient visit: %s", err.Error())
	} else {
		diagnosisQuestionId = qi.Id
	}
	if qi, err := testData.DataApi.GetQuestionInfo("q_acne_severity", 1); err != nil {
		t.Fatalf("Unable to get the questionIds for the question tags requested for the doctor to diagnose patient visit: %s", err.Error())
	} else {
		severityQuestionId = qi.Id
	}
	if qi, err := testData.DataApi.GetQuestionInfo("q_acne_type", 1); err != nil {
		t.Fatalf("Unable to get the questionIds for the question tags requested for the doctor to diagnose patient visit: %s", err.Error())
	} else {
		acneTypeQuestionId = qi.Id
	}

	diagnosePatientHandler := apiservice.NewDiagnosePatientHandler(testData.DataApi, testData.AuthApi, testData.CloudStorageService)
	ts := httptest.NewServer(diagnosePatientHandler)
	defer ts.Close()

	answerToQuestionItem := &apiservice.AnswerToQuestionItem{}
	answerToQuestionItem.QuestionId = diagnosisQuestionId
	answerToQuestionItem.AnswerIntakes = []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerId: 102}}

	answerToQuestionItem2 := &apiservice.AnswerToQuestionItem{}
	answerToQuestionItem2.QuestionId = severityQuestionId
	answerToQuestionItem2.AnswerIntakes = []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerId: 107}}

	answerToQuestionItem3 := &apiservice.AnswerToQuestionItem{}
	answerToQuestionItem3.QuestionId = acneTypeQuestionId
	answerToQuestionItem3.AnswerIntakes = []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerId: 109}}

	answerIntakeRequestBody.Questions = []*apiservice.AnswerToQuestionItem{answerToQuestionItem, answerToQuestionItem2, answerToQuestionItem3}

	requestData, err := json.Marshal(answerIntakeRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body")
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(requestData), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to successfully submit the diagnosis of a patient visit: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of response on submitting diagnosis of patient visit : " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to submit diagnosis of patient visit "+string(body), t)
	return
}

func StartReviewingPatientVisit(PatientVisitId int64, doctor *common.Doctor, testData TestData, t *testing.T) *apiservice.DoctorPatientVisitReviewResponse {
	doctorPatientVisitReviewHandler := &apiservice.DoctorPatientVisitReviewHandler{DataApi: testData.DataApi, LayoutStorageService: testData.CloudStorageService, PatientPhotoStorageService: testData.CloudStorageService}
	ts := httptest.NewServer(doctorPatientVisitReviewHandler)
	defer ts.Close()

	resp, err := authGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(PatientVisitId, 10), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review for patient: " + err.Error())
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse body of response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get patient visit review: "+string(respBody), t)

	doctorPatientVisitReviewResponse := &apiservice.DoctorPatientVisitReviewResponse{}
	err = json.Unmarshal(respBody, doctorPatientVisitReviewResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body in to json object: " + err.Error())
	}
	return doctorPatientVisitReviewResponse
}

func SubmitPatientVisitBackToPatient(PatientVisitId int64, doctor *common.Doctor, testData TestData, t *testing.T) {
	doctorSubmitPatientVisitReviewHandler := &apiservice.DoctorSubmitPatientVisitReviewHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(doctorSubmitPatientVisitReviewHandler)
	defer ts.Close()

	resp, err := authPost(ts.URL, "application/x-www-form-urlencoded", bytes.NewBufferString("patient_visit_id="+strconv.FormatInt(PatientVisitId, 10)), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to close patient visit " + err.Error())
	}
	CheckSuccessfulStatusCode(resp, "Unable to make successful call to close the patient visit", t)
}
