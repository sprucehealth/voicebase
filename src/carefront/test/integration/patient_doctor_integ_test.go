package integration

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_worker"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/aws/sqs"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"carefront/patient_treatment_plan"

	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/samuel/go-metrics/metrics"
)

func TestPatientVisitReview(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from id: " + err.Error())
	}

	patientVisitResponse, _ := signupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf("Unable to get patient from patient visit info: %s", err)
	}

	pharmacySelection := &pharmacy.PharmacyData{
		SourceId:     "12345",
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		AddressLine1: "12345 Marin Street",
		City:         "San Francisco",
		State:        "CA",
		Phone:        "12345667",
	}

	if err := testData.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), pharmacySelection); err != nil {
		t.Fatalf("Unable to update pharmacy for patient %s", err)
	}

	// try getting the patient visit review for this patient visit and it should fail
	patientVisitReviewHandler := &patient_treatment_plan.PatientVisitReviewHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(patientVisitReviewHandler)
	defer ts.Close()

	resp, err := AuthGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make call to get the patient visit review for patient visit: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected to get %d for call to get patient visit review but instead got %d", http.StatusBadRequest, resp.StatusCode)
	}

	// once the doctor has started reviewing the case, lets go ahead and get the doctor to close the case with no diagnosis
	stubErxService := &erx.StubErxService{}
	stubErxService.PatientErxId = 10
	stubErxService.PrescriptionIdsToReturn = []int64{}
	stubErxService.PrescriptionIdToPrescriptionStatuses = make(map[int64][]common.StatusEvent)
	stubErxService.PharmacyToSendPrescriptionTo = pharmacySelection.SourceId

	erxStatusQueue := &common.SQSQueue{}
	erxStatusQueue.QueueService = &sqs.StubSQS{}
	erxStatusQueue.QueueUrl = "local-erx"
	doctorSubmitPatientVisitReviewHandler := &apiservice.DoctorSubmitPatientVisitReviewHandler{
		DataApi:        testData.DataApi,
		ERxApi:         stubErxService,
		ErxStatusQueue: erxStatusQueue,
		ERxRouting:     true,
	}
	ts3 := httptest.NewServer(doctorSubmitPatientVisitReviewHandler)
	defer ts3.Close()

	resp, err = AuthPost(ts3.URL, "application/x-www-form-urlencoded", bytes.NewBufferString("patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10)), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make call to close patient visit " + err.Error())
	}
	CheckSuccessfulStatusCode(resp, "Unable to make successful call to close the patient visit", t)

	// start a new patient visit
	patientVisitResponse, treatmentPlan := signupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)
	patient, err = testData.DataApi.GetPatientFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf("Unable to get patient from patient visit id: %s", err)
	}

	if err := testData.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), pharmacySelection); err != nil {
		t.Fatalf("Unable to update pharmacy for patient %s", err)
	}
	//
	//
	// SUBMIT DIAGNOSIS
	//
	//

	SubmitPatientVisitDiagnosis(patientVisitResponse.PatientVisitId, doctor, testData, t)

	//
	//
	// SUBMIT TREATMENT PLAN
	//
	//
	// doctor now attempts to add a couple treatments for patient
	treatment1 := &common.Treatment{
		DrugInternalName: "Advil",
		DosageStrength:   "10 mg",
		DispenseValue:    1,
		DispenseUnitId:   encoding.NewObjectId(26),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		SubstitutionsAllowed: true,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		OTC:                 true,
		PharmacyNotes:       "testing pharmacy notes",
		PatientInstructions: "patient instructions",
		DrugDBIds: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	treatment2 := &common.Treatment{
		DrugInternalName: "Advil 2",
		DosageStrength:   "100 mg",
		DispenseValue:    2,
		DispenseUnitId:   encoding.NewObjectId(27),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 3,
		},
		SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 12,
		},
		OTC:                 false,
		PharmacyNotes:       "testing pharmacy notes 2",
		PatientInstructions: "patient instructions 2",
		DrugDBIds: map[string]string{
			"drug_db_id_3": "12414",
			"drug_db_id_4": "214",
		},
	}

	treatments := []*common.Treatment{treatment1, treatment2}

	stubErxService.PrescriptionIdsToReturn = []int64{10, 20}
	stubErxService.PrescriptionIdToPrescriptionStatuses[10] = []common.StatusEvent{common.StatusEvent{Status: api.ERX_STATUS_SENT}}
	stubErxService.PrescriptionIdToPrescriptionStatuses[20] = []common.StatusEvent{common.StatusEvent{Status: api.ERX_STATUS_ERROR, StatusDetails: "error test"}}

	getTreatmentsResponse := addAndGetTreatmentsForPatientVisit(testData, treatments, doctor.AccountId.Int64(), patientVisitResponse.PatientVisitId, t)
	if len(getTreatmentsResponse.TreatmentList.Treatments) != 2 {
		t.Fatalf("Expected 2 treatments to be returned, instead got back %d", len(getTreatmentsResponse.TreatmentList.Treatments))
	}

	//
	//
	// SUBMIT REGIMEN PLAN
	//
	//
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)

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
	getRegimenPlanResponse := getRegimenPlanForPatientVisit(testData, doctor, patientVisitResponse.PatientVisitId, t)
	if len(getRegimenPlanResponse.RegimenSections) != 2 {
		t.Fatal("Expected 2 regimen sections")
	}

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
	doctorAdviceRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)

	doctorAdviceResponse := updateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	validateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)
	getAdviceResponse := getAdvicePointsInPatientVisit(testData, doctor, patientVisitResponse.PatientVisitId, t)
	if len(getAdviceResponse.SelectedAdvicePoints) != len(doctorAdviceRequest.AllAdvicePoints) {
		t.Fatal("Expected number of advice points not returned")
	}
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
	resp, err = AuthPost(ts5.URL, "application/x-www-form-urlencoded", bytes.NewBufferString(requestBody), doctor.AccountId.Int64())
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
	resp, err = AuthPost(ts3.URL, "application/x-www-form-urlencoded", bytes.NewBufferString("patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10)), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make call to close patient visit " + err.Error())
	}
	CheckSuccessfulStatusCode(resp, "Unable to make successful call to close the patient visit", t)

	// get an updated view of the patient informatio nfrom the database again given that weve assigned a prescription id to him
	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from database: " + err.Error())
	}

	prescriptionStatuses, err := testData.DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get prescription statuses for patient: " + err.Error())
	}
	// there should be a total of 4 prescription statuses for this patient, with 2 per treatment
	if len(prescriptionStatuses) != 2 {
		t.Fatalf("Expected there to be 1 status events per treatment, instead have a total of %d", len(prescriptionStatuses))
	}

	for _, status := range prescriptionStatuses {
		if status.Status != api.ERX_STATUS_SENDING {
			t.Fatal("Expected the prescription status to be either eRxSent or Sending")
		}
	}

	// attempt to consume the message put into the queue
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxService, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	prescriptionStatuses, err = testData.DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get prescription statuses for patient: " + err.Error())
	}

	// there should be a total of 2 prescription statuses for this patient, with 1 per treatment
	if len(prescriptionStatuses) != 2 {
		t.Fatalf("Expected there to be 1 status events per treatment, instead have a total of %d", len(prescriptionStatuses))
	}

	for _, status := range prescriptionStatuses {
		if status.ItemId == 20 && (status.Status != api.ERX_STATUS_ERROR || status.Status != api.ERX_STATUS_SENDING) {
			t.Fatal("Expected the prescription status to be error for 1 treatment")
		}

		if status.Status != api.ERX_STATUS_SENT && status.Status != api.ERX_STATUS_SENDING && status.Status != api.ERX_STATUS_ERROR {
			t.Fatal("Expected the prescription status to be either eRxSent, Sending, or Error")
		}
	}

	treatments, err = testData.DataApi.GetTreatmentsForPatient(patient.PatientId.Int64())
	if err != nil {
		t.Fatalf(err.Error())
	}

	if len(treatments) != 2 {
		t.Fatal("Expected 2 treatments to be returned within treatment plan")
	}

	for _, treatment := range treatments {
		if treatment.Id.Int64() == 20 && (treatment.ERx.RxHistory[0].Status != api.ERX_STATUS_ERROR) {
			t.Fatal("Expected the prescription status to be error for 1 treatment")
		}

		if treatment.ERx.RxHistory[0].Status != api.ERX_STATUS_SENT && treatment.ERx.RxHistory[0].Status != api.ERX_STATUS_SENDING && treatment.ERx.RxHistory[0].Status != api.ERX_STATUS_ERROR {
			t.Fatalf("Expected the prescription status to be either eRxSent, Sending, or Error. Instead it is %s", treatment.ERx.RxHistory[0].Status)
		}
	}

	//
	//
	// GET PATIENT VISIT REVIEW FOR PATIENT
	//
	//
	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get the patient object given the id: " + err.Error())
	}
	resp, err = AuthGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review: " + err.Error())
	}

	diagnosisSummary, err := testData.DataApi.GetDiagnosisSummaryForTreatmentPlan(treatmentPlan.Id.Int64())
	if err != nil {
		t.Fatalf("Error while getting diagnosis summary %s", err)
	}

	if diagnosisSummary == nil || diagnosisSummary.Summary == "" {
		t.Fatalf("Diagnosis summary expected to exist")
	}
}
