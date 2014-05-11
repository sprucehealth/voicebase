package integration

import (
	"bytes"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/erx"
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"strconv"
	"testing"
)

func getRegimenPlanForPatientVisit(testData TestData, doctor *common.Doctor, patientVisitId int64, t *testing.T) *common.RegimenPlan {
	doctorTreatmentPlanHandler := &apiservice.DoctorTreatmentPlanHandler{
		DataApi: testData.DataApi,
	}
	ts := httptest.NewServer(doctorTreatmentPlanHandler)
	defer ts.Close()

	resp, err := AuthGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitId, 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get regimen for patient visit: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response for getting the regimen plan: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get regimen plan for patient visit: "+string(body), t)

	doctorTreatmentPlanResponse := &apiservice.DoctorTreatmentPlanResponse{}
	err = json.Unmarshal(body, doctorTreatmentPlanResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal body into json object: " + err.Error())
	}

	return doctorTreatmentPlanResponse.TreatmentPlan.RegimenPlan
}

func createRegimenPlanForPatientVisit(doctorRegimenRequest *common.RegimenPlan, testData TestData, doctor *common.Doctor, t *testing.T) *common.RegimenPlan {
	doctorRegimenHandler := apiservice.NewDoctorRegimenHandler(testData.DataApi)
	ts := httptest.NewServer(doctorRegimenHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(doctorRegimenRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding regimen steps: " + err.Error())
	}

	resp, err := AuthPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to create regimen for patient visit")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of response after making call to create regimen plan")
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to create regimen plan for patient: "+string(body), t)

	regimenPlanResponse := &common.RegimenPlan{}
	err = json.Unmarshal(body, regimenPlanResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into json object : " + err.Error())
	}

	return regimenPlanResponse
}

func getAdvicePointsInPatientVisit(testData TestData, doctor *common.Doctor, patientVisitId int64, t *testing.T) *common.Advice {
	doctorTreatmentPlanHandler := &apiservice.DoctorTreatmentPlanHandler{
		DataApi: testData.DataApi,
	}
	ts := httptest.NewServer(doctorTreatmentPlanHandler)
	defer ts.Close()

	resp, err := AuthGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitId, 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get advice points for patient visit: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response for getting the advice points: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful call to get advice points for patient visit : "+string(body), t)

	doctorTreatmentPlanResponse := &apiservice.DoctorTreatmentPlanResponse{}
	err = json.Unmarshal(body, doctorTreatmentPlanResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response body into the advice repsonse object: " + err.Error())
	}

	return doctorTreatmentPlanResponse.TreatmentPlan.Advice
}

func updateAdvicePointsForPatientVisit(doctorAdviceRequest *common.Advice, testData TestData, doctor *common.Doctor, t *testing.T) *common.Advice {
	doctorAdviceHandler := apiservice.NewDoctorAdviceHandler(testData.DataApi)
	ts := httptest.NewServer(doctorAdviceHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(doctorAdviceRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding advice points: " + err.Error())
	}

	resp, err := AuthPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to add advice points to patient visit " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable tp read body of the response after adding advice points to patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to add advice points : "+string(body), t)

	doctorAdviceResponse := &common.Advice{}
	err = json.Unmarshal(body, doctorAdviceResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body into json object : " + err.Error())
	}

	return doctorAdviceResponse
}

func addAndGetTreatmentsForPatientVisit(testData TestData, treatments []*common.Treatment, doctorAccountId, PatientVisitId int64, t *testing.T) *apiservice.GetTreatmentsResponse {
	stubErxApi := &erx.StubErxService{
		SelectedMedicationToReturn: &common.Treatment{},
	}

	treatmentRequestBody := apiservice.AddTreatmentsRequestBody{PatientVisitId: encoding.NewObjectId(PatientVisitId), Treatments: treatments}
	treatmentsHandler := &apiservice.TreatmentsHandler{
		DataApi: testData.DataApi,
		ErxApi:  stubErxApi,
	}

	ts := httptest.NewServer(treatmentsHandler)
	defer ts.Close()

	data, err := json.Marshal(&treatmentRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err := AuthPost(ts.URL, "application/json", bytes.NewBuffer(data), doctorAccountId)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}

	addTreatmentsResponse := &apiservice.GetTreatmentsResponse{}
	err = json.NewDecoder(resp.Body).Decode(addTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call made to add treatments for patient visit: ", t)

	if addTreatmentsResponse.TreatmentList == nil || len(addTreatmentsResponse.TreatmentList.Treatments) == 0 {
		t.Fatal("Treatment ids expected to be returned for the treatments just added")
	}

	return addTreatmentsResponse
}
