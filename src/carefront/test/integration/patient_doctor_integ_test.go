package integration

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_worker"
	"carefront/common"
	"carefront/libs/aws/sqs"
	"carefront/libs/erx"
	"github.com/samuel/go-metrics/metrics"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestPatientVisitReview(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := CreatePatientVisitForPatient(signedupPatientResponse.Patient.PatientId, testData, t)
	SubmitPatientVisitForPatient(signedupPatientResponse.Patient.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId)
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	// try getting the patient visit review for this patient visit and it should fail
	patientVisitReviewHandler := &apiservice.PatientVisitReviewHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(patientVisitReviewHandler)
	defer ts.Close()

	resp, err := authGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), patient.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get the patient visit review for patient visit: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected to get %d for call to get patient visit review but instead got %d", http.StatusBadRequest, resp.StatusCode)
	}

	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from id: " + err.Error())
	}

	// now lets go ahead and get doctor to start reviewing the patient visit and then submit the patient visit review
	doctorPatientVisitReviewHandler := &apiservice.DoctorPatientVisitReviewHandler{DataApi: testData.DataApi, LayoutStorageService: testData.CloudStorageService, PatientPhotoStorageService: testData.CloudStorageService}
	ts2 := httptest.NewServer(doctorPatientVisitReviewHandler)
	defer ts2.Close()
	resp, err = authGet(ts2.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review for patient: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call for doctor to start reviewing patient visit", t)

	// once the doctor has started reviewing the case, lets go ahead and get the doctor to close the case with no diagnosis

	stubErxService := &erx.StubErxService{}
	stubErxService.PatientErxId = 10
	stubErxService.PrescriptionIdsToReturn = []int64{}
	stubErxService.PrescriptionIdToPrescriptionStatus = make(map[int64]string)

	erxStatusQueue := &common.SQSQueue{}
	erxStatusQueue.QueueService = &sqs.StubSQS{}
	erxStatusQueue.QueueUrl = api.ERX_STATUS_QUEUE
	doctorSubmitPatientVisitReviewHandler := &apiservice.DoctorSubmitPatientVisitReviewHandler{
		DataApi:        testData.DataApi,
		ERxApi:         stubErxService,
		ErxStatusQueue: erxStatusQueue,
		ERxRouting:     true,
	}
	ts3 := httptest.NewServer(doctorSubmitPatientVisitReviewHandler)
	defer ts3.Close()

	resp, err = authPost(ts3.URL, "application/x-www-form-urlencoded", bytes.NewBufferString("patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10)), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to close patient visit " + err.Error())
	}
	CheckSuccessfulStatusCode(resp, "Unable to make successful call to close the patient visit", t)

	fromTime := time.Now().Add(-24 * time.Hour).Unix()
	toTime := time.Now().Unix()
	doctorPrescriptionsResponse := getPrescriptionsForDoctor(testData.DataApi, t, doctor, fromTime, toTime)

	if len(doctorPrescriptionsResponse.TreatmentPlans) > 0 {
		t.Fatal("Expected number of treatment plans to be 0")
	}

	// now, lets try and get the patient visit review again
	resp, err = authGet(ts2.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review: " + err.Error())
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse body of response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get patient visit review: "+string(respBody), t)

	patientVisitReviewResponse := &apiservice.PatientVisitReviewResponse{}
	err = json.Unmarshal(respBody, patientVisitReviewResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body in to json object: " + err.Error())
	}

	if patientVisitReviewResponse.DiagnosisSummary != nil || patientVisitReviewResponse.Treatments != nil || patientVisitReviewResponse.RegimenPlan != nil ||
		patientVisitReviewResponse.Advice != nil || patientVisitReviewResponse.Followup != nil {
		t.Fatal("Expected there to exist no review for this patient visit, but some parts of it do exist")
	}

	patientVisitResponse = GetPatientVisitForPatient(patient.PatientId, testData, t)

	// start a new patient visit
	signedupPatientResponse = SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse = CreatePatientVisitForPatient(signedupPatientResponse.Patient.PatientId, testData, t)
	SubmitPatientVisitForPatient(signedupPatientResponse.Patient.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	// get doctor to start reviewing it
	StartReviewingPatientVisit(patientVisitResponse.PatientVisitId, doctor, testData, t)

	//
	//
	// SUBMIT DIAGNOSIS
	//
	//

	submitPatientVisitDiagnosis(patientVisitResponse.PatientVisitId, doctor, testData, t)

	//
	//
	// SUBMIT TREATMENT PLAN
	//
	//
	// doctor now attempts to add a couple treatments for patient
	treatment1 := &common.Treatment{}
	treatment1.DrugInternalName = "Advil"
	treatment1.DosageStrength = "10 mg"
	treatment1.DispenseValue = 1
	treatment1.DispenseUnitId = 26
	treatment1.NumberRefills = 1
	treatment1.SubstitutionsAllowed = true
	treatment1.DaysSupply = 1
	treatment1.OTC = true
	treatment1.PharmacyNotes = "testing pharmacy notes"
	treatment1.PatientInstructions = "patient instructions"
	drugDBIds := make(map[string]string)
	drugDBIds["drug_db_id_1"] = "12315"
	drugDBIds["drug_db_id_2"] = "124"
	treatment1.DrugDBIds = drugDBIds

	treatment2 := &common.Treatment{}
	treatment2.DrugInternalName = "Advil 2"
	treatment2.DosageStrength = "100 mg"
	treatment2.DispenseValue = 2
	treatment2.DispenseUnitId = 27
	treatment2.NumberRefills = 3
	treatment2.SubstitutionsAllowed = false
	treatment2.DaysSupply = 12
	treatment2.OTC = false
	treatment2.PharmacyNotes = "testing pharmacy notes 2"
	treatment2.PatientInstructions = "patient instructions 2"
	drugDBIds = make(map[string]string)
	drugDBIds["drug_db_id_3"] = "12414"
	drugDBIds["drug_db_id_4"] = "214"
	treatment2.DrugDBIds = drugDBIds

	treatments := []*common.Treatment{treatment1, treatment2}

	stubErxService.PrescriptionIdsToReturn = []int64{10, 20}
	stubErxService.PrescriptionIdToPrescriptionStatus[10] = api.ERX_STATUS_SENT
	stubErxService.PrescriptionIdToPrescriptionStatus[20] = api.ERX_STATUS_ERROR

	addAndGetTreatmentsForPatientVisit(testData, treatments, doctor.AccountId, patientVisitResponse.PatientVisitId, t)

	//
	//
	// SUBMIT REGIMEN PLAN
	//
	//
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.PatientVisitId = patientVisitResponse.PatientVisitId

	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED

	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED
	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}

	regimenSection := &common.RegimenSection{}
	regimenSection.RegimenName = "morning"
	regimenSection.RegimenSteps = []*common.DoctorInstructionItem{regimenPlanRequest.AllRegimenSteps[0]}

	regimenSection2 := &common.RegimenSection{}
	regimenSection2.RegimenName = "night"
	regimenSection2.RegimenSteps = []*common.DoctorInstructionItem{regimenPlanRequest.AllRegimenSteps[1]}

	regimenPlanRequest.RegimenSections = []*common.RegimenSection{regimenSection, regimenSection2}
	regimenPlanResponse := createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	//
	//
	// SUBMIT ADVICE
	//
	//
	// lets go ahead and add a couple of advice points
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{advicePoint1, advicePoint2}
	doctorAdviceRequest.SelectedAdvicePoints = doctorAdviceRequest.AllAdvicePoints
	doctorAdviceRequest.PatientVisitId = patientVisitResponse.PatientVisitId

	doctorAdviceResponse := updateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	validateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	//
	//
	// SUBMIT FOLLOW UP
	//
	//

	// lets add a follow up time for 1 week from now
	doctorFollowupHandler := apiservice.NewPatientVisitFollowUpHandler(testData.DataApi)
	ts5 := httptest.NewServer(doctorFollowupHandler)
	defer ts5.Close()

	requestBody := fmt.Sprintf("patient_visit_id=%d&follow_up_unit=week&follow_up_value=1", patientVisitResponse.PatientVisitId)
	resp, err = authPost(ts5.URL, "application/x-www-form-urlencoded", bytes.NewBufferString(requestBody), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make successful call to add follow up time for patient visit: " + err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read the response body: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to add follow up for patient visit: "+string(body), t)

	//
	//
	// SUBMIT VISIT FOR PATIENT VISIT REVIEW
	//
	//

	// get doctor to submit the patient visit review
	resp, err = authPost(ts3.URL, "application/x-www-form-urlencoded", bytes.NewBufferString("patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10)), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to close patient visit " + err.Error())
	}
	CheckSuccessfulStatusCode(resp, "Unable to make successful call to close the patient visit", t)

	// number of prescripitons returned when not including the time at which it was submitted should be 0
	doctorPrescriptionsResponse = getPrescriptionsForDoctor(testData.DataApi, t, doctor, fromTime, toTime)
	if len(doctorPrescriptionsResponse.TreatmentPlans) > 0 {
		t.Fatal("Expected there to be no prescriptions for this doctor yet")
	}

	// now, lets try again while including time
	toTime = time.Now().Add(10 * time.Minute).Unix()
	doctorPrescriptionsResponse = getPrescriptionsForDoctor(testData.DataApi, t, doctor, fromTime, toTime)
	if len(doctorPrescriptionsResponse.TreatmentPlans) != 1 {
		t.Fatalf("Expected there to be 1 treatment plan for this doctor, instead we have %d", len(doctorPrescriptionsResponse.TreatmentPlans))
	}

	if len(doctorPrescriptionsResponse.TreatmentPlans[0].Treatments) != 2 {
		t.Fatalf("Expected there to be 2 treatments in this treatment plan for this doctor, instead we have %d", len(doctorPrescriptionsResponse.TreatmentPlans[0].Treatments))
	}

	// lets get the patient object from database to get an updated view of the object
	patient, err = testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId)
	if err != nil {
		t.Fatal("Unable to get patient from database: " + err.Error())
	}

	prescriptionStatuses, err := testData.DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId)
	if err != nil {
		t.Fatal("Unable to get prescription statuses for patient: " + err.Error())
	}
	// there should be a total of 4 prescription statuses for this patient, with 2 per treatment
	if len(prescriptionStatuses) != 2 {
		t.Fatalf("Expected there to be 1 status events per treatment, instead have a total of %d", len(prescriptionStatuses))
	}

	for _, status := range prescriptionStatuses {
		if status.PrescriptionStatus != api.ERX_STATUS_SENDING {
			t.Fatal("Expected the prescription status to be either eRxSent or Sending")
		}
	}

	// attempt to consume the message put into the queue
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxService, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	prescriptionStatuses, err = testData.DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId)
	if err != nil {
		t.Fatal("Unable to get prescription statuses for patient: " + err.Error())
	}

	// there should be a total of 2 prescription statuses for this patient, with 1 per treatment
	if len(prescriptionStatuses) != 2 {
		t.Fatalf("Expected there to be 1 status events per treatment, instead have a total of %d", len(prescriptionStatuses))
	}

	for _, status := range prescriptionStatuses {
		if status.TreatmentId == 20 && (status.PrescriptionStatus != api.ERX_STATUS_ERROR || status.PrescriptionStatus != api.ERX_STATUS_SENDING) {
			t.Fatal("Expected the prescription status to be error for 1 treatment")
		}

		if status.PrescriptionStatus != api.ERX_STATUS_SENT && status.PrescriptionStatus != api.ERX_STATUS_SENDING && status.PrescriptionStatus != api.ERX_STATUS_ERROR {
			t.Fatal("Expected the prescription status to be either eRxSent, Sending, or Error")
		}
	}

	// number of prescriptions returned for doctor even after the prescription status update should be 2 total
	doctorPrescriptionsResponse = getPrescriptionsForDoctor(testData.DataApi, t, doctor, fromTime, toTime)
	if len(doctorPrescriptionsResponse.TreatmentPlans) != 1 {
		t.Fatal("Expected 1 treatment plan to be returned")
	}

	if len(doctorPrescriptionsResponse.TreatmentPlans[0].Treatments) != 2 {
		t.Fatal("Expected 2 treatments to be returned within treatment plan")
	}

	for _, treatment := range doctorPrescriptionsResponse.TreatmentPlans[0].Treatments {
		if treatment.Id == 20 && (treatment.PrescriptionStatus != api.ERX_STATUS_ERROR || treatment.PrescriptionStatus != api.ERX_STATUS_SENDING) {
			t.Fatal("Expected the prescription status to be error for 1 treatment")
		}

		if treatment.PrescriptionStatus != api.ERX_STATUS_SENT && treatment.PrescriptionStatus != api.ERX_STATUS_SENDING && treatment.PrescriptionStatus != api.ERX_STATUS_ERROR {
			t.Fatal("Expected the prescription status to be either eRxSent, Sending, or Error")
		}
	}

	//
	//
	// GET PATIENT VISIT REVIEW FOR PATIENT
	//
	//
	patient, err = testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId)
	if err != nil {
		t.Fatal("Unable to get the patient object given the id: " + err.Error())
	}
	resp, err = authGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), patient.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review: " + err.Error())
	}

	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse body of response: " + err.Error())

	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get patient visit review: "+string(respBody), t)

	patientVisitReviewResponse = &apiservice.PatientVisitReviewResponse{}
	err = json.Unmarshal(respBody, patientVisitReviewResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body in to json object: " + err.Error())
	}

	patientVisitResponse = GetPatientVisitForPatient(patient.PatientId, testData, t)

	if patientVisitReviewResponse.DiagnosisSummary == nil || patientVisitReviewResponse.Treatments == nil || patientVisitReviewResponse.RegimenPlan == nil ||
		patientVisitReviewResponse.Advice == nil || patientVisitReviewResponse.Followup == nil {
		t.Fatal("Expected there to exist all sections of the review")
	}
}

func getPrescriptionsForDoctor(dataApi api.DataAPI, t *testing.T, doctor *common.Doctor, fromTime, toTime int64) *apiservice.DoctorPrescriptionsResponse {
	// check the prescriptions the doctor has sent in and there should be none
	doctorPrescriptionsHandler := &apiservice.DoctorPrescriptionsHandler{
		DataApi: dataApi,
	}
	ts4 := httptest.NewServer(doctorPrescriptionsHandler)
	defer ts4.Close()

	resp, err := authGet(ts4.URL+fmt.Sprintf("?from=%d&to=%d", fromTime, toTime), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get prescriptions for doctor: " + err.Error())
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}

	doctorPrescriptionsResponse := &apiservice.DoctorPrescriptionsResponse{}
	err = json.Unmarshal(respBody, doctorPrescriptionsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body into json object: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to get prescriptions for doctor "+string(respBody), t)
	return doctorPrescriptionsResponse
}
