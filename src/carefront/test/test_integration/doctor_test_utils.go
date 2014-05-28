package test_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/patient_file"
	thriftapi "carefront/thrift/api"
)

func signupRandomTestDoctor(t *testing.T, dataApi api.DataAPI, authApi thriftapi.Auth) (signedupDoctorResponse *apiservice.DoctorSignedupResponse, email, password string) {
	authHandler := &apiservice.SignupDoctorHandler{AuthApi: authApi, DataApi: dataApi}
	ts := httptest.NewServer(authHandler)
	defer ts.Close()

	email = strconv.FormatInt(time.Now().UnixNano(), 10) + "@example.com"
	password = "12345"
	params := &url.Values{}
	params.Set("first_name", "Test")
	params.Set("last_name", "Test")
	params.Set("email", email)
	params.Set("password", password)
	params.Set("dob", "1987-11-08")
	params.Set("gender", "male")
	params.Set("clinician_id", os.Getenv("DOSESPOT_USER_ID"))
	params.Set("phone", "123451616")
	params.Set("address_line_1", "12345 Main street")
	params.Set("address_line_2", "apt 11415")
	params.Set("city", "san francisco")
	params.Set("state", "ca")
	params.Set("zip_code", "94115")

	res, err := AuthPost(ts.URL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()), 0)
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
	clinicId, _ := strconv.ParseInt(os.Getenv("DOSESPOT_CLINIC_ID"), 10, 64)
	userId, _ := strconv.ParseInt(os.Getenv("DOSESPOT_USER_ID"), 10, 64)

	if clinicKey == "" {
		t.Log("WARNING: skipping doctor drug search test since the dosespot ids are not present as environment variables")
		t.SkipNow()
	}

	erx := erx.NewDoseSpotService(clinicId, userId, clinicKey, nil)
	return erx
}

func SubmitPatientVisitDiagnosis(PatientVisitId int64, doctor *common.Doctor, testData TestData, t *testing.T) (diagnosisQuestionId, severityQuestionId, acneTypeQuestionId int64) {
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

	resp, err := AuthPost(ts.URL, "application/json", bytes.NewBuffer(requestData), doctor.AccountId.Int64())
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

func StartReviewingPatientVisit(patientVisitId int64, doctor *common.Doctor, testData TestData, t *testing.T) {
	doctorPatientVisitReviewHandler := patient_file.NewDoctorPatientVisitReviewHandler(testData.DataApi, nil, testData.CloudStorageService, testData.CloudStorageService)

	ts := httptest.NewServer(doctorPatientVisitReviewHandler)
	defer ts.Close()

	resp, err := AuthGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitId, 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review for patient: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get patient visit review: ", t)
}

func pickATreatmentPlanForPatientVisit(patientVisitId int64, doctor *common.Doctor, favoriteTreatmentPlan *common.FavoriteTreatmentPlan, testData TestData, t *testing.T) *apiservice.DoctorTreatmentPlanResponse {
	doctorPickTreatmentPlanHandler := &apiservice.DoctorTreatmentPlanHandler{
		DataApi: testData.DataApi,
	}

	ts := httptest.NewServer(doctorPickTreatmentPlanHandler)
	defer ts.Close()

	params := url.Values{}
	params.Set("patient_visit_id", strconv.FormatInt(patientVisitId, 10))
	if favoriteTreatmentPlan != nil {
		params.Set("dr_favorite_treatment_plan_id", strconv.FormatInt(favoriteTreatmentPlan.Id.Int64(), 10))
	}

	resp, err := AuthPut(ts.URL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()), doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to pick a treatment plan for the patient visit doctor %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected successful picking up of treatment plan instead got %d", resp.StatusCode)
	}

	responseData := &apiservice.DoctorTreatmentPlanResponse{}
	if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into response json: %s", err)
	}

	return responseData
}

func SubmitPatientVisitBackToPatient(patientVisitId int64, doctor *common.Doctor, testData TestData, t *testing.T) {
	doctorSubmitPatientVisitReviewHandler := &apiservice.DoctorSubmitPatientVisitReviewHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(doctorSubmitPatientVisitReviewHandler)
	defer ts.Close()

	resp, err := AuthPost(ts.URL, "application/x-www-form-urlencoded", bytes.NewBufferString("patient_visit_id="+strconv.FormatInt(patientVisitId, 10)), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make call to close patient visit " + err.Error())
	}
	CheckSuccessfulStatusCode(resp, "Unable to make successful call to close the patient visit", t)
}
