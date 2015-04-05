package test_integration

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/test"
)

func TestPatientVisitReview(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from id: " + err.Error())
	}

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatalf("Unable to get patient from patient visit info: %s", err)
	}

	pharmacySelection := &pharmacy.PharmacyData{
		SourceID:     12345,
		Source:       pharmacy.PharmacySourceSurescripts,
		AddressLine1: "12345 Marin Street",
		City:         "San Francisco",
		State:        "CA",
		Phone:        "12345667",
	}

	if err := testData.DataAPI.UpdatePatientPharmacy(patient.PatientID.Int64(), pharmacySelection); err != nil {
		t.Fatalf("Unable to update pharmacy for patient %s", err)
	}

	// try getting the patient visit review for this patient visit and it should fail

	resp, err := testData.AuthGet(testData.APIServer.URL+apipaths.TreatmentPlanURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlan.ID.Int64(), 10), patient.AccountID.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusNotFound, resp.StatusCode)

	// once the doctor has started reviewing the case, lets go ahead and get the doctor to close the case with no diagnosis
	stubErxService := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxService.PatientErxID = 10
	stubErxService.PrescriptionIdsToReturn = []int64{}
	stubErxService.PrescriptionIDToPrescriptionStatuses = make(map[int64][]common.StatusEvent)
	stubErxService.PharmacyToSendPrescriptionTo = pharmacySelection.SourceID

	SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)
	// consume the message
	doctor_treatment_plan.StartWorker(testData.DataAPI, stubErxService, testData.Config.Dispatcher, testData.Config.ERxRoutingQueue, testData.Config.ERxStatusQueue, 0, metrics.NewRegistry())

	// start a new patient visit
	patientVisitResponse, treatmentPlan = CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err = testData.DataAPI.GetPatientFromPatientVisitID(patientVisitResponse.PatientVisitID)
	test.OK(t, err)

	err = testData.DataAPI.UpdatePatientPharmacy(patient.PatientID.Int64(), pharmacySelection)
	test.OK(t, err)

	//
	//
	// SUBMIT DIAGNOSIS
	//
	//

	SubmitPatientVisitDiagnosis(patientVisitResponse.PatientVisitID, doctor, testData, t)

	//
	//
	// SUBMIT TREATMENT PLAN
	//
	//
	// doctor now attempts to add a couple treatments for patient
	treatment1 := &common.Treatment{
		DrugInternalName: "Drug1 (Route1 - Form1)",
		DosageStrength:   "Strength1",
		DispenseValue:    1,
		DispenseUnitID:   encoding.NewObjectID(26),
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
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	treatment2 := &common.Treatment{
		DrugInternalName: "Drug2 (Route2 - Form2)",
		DosageStrength:   "Strength2",
		DispenseValue:    2,
		DispenseUnitID:   encoding.NewObjectID(27),
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
		DrugDBIDs: map[string]string{
			"drug_db_id_3": "12414",
			"drug_db_id_4": "214",
		},
	}

	treatments := []*common.Treatment{treatment1, treatment2}

	getTreatmentsResponse := AddAndGetTreatmentsForPatientVisit(testData, treatments, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)
	if len(getTreatmentsResponse.TreatmentList.Treatments) != 2 {
		t.Fatalf("Expected 2 treatments to be returned, instead got back %d", len(getTreatmentsResponse.TreatmentList.Treatments))
	}

	//
	//
	// SUBMIT REGIMEN PLAN
	//
	//
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: treatmentPlan.ID,
	}

	regimenStep1 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 1",
		State: common.StateAdded,
	}

	regimenStep2 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 2",
		State: common.StateAdded,
	}
	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}

	regimenSection := &common.RegimenSection{
		Name:  "morning",
		Steps: []*common.DoctorInstructionItem{regimenPlanRequest.AllSteps[0]},
	}

	regimenSection2 := &common.RegimenSection{
		Name:  "night",
		Steps: []*common.DoctorInstructionItem{regimenPlanRequest.AllSteps[1]},
	}

	regimenPlanRequest.Sections = []*common.RegimenSection{regimenSection, regimenSection2}
	regimenPlanResponse := CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)
	getRegimenPlanResponse := GetRegimenPlanForTreatmentPlan(testData, doctor, treatmentPlan.ID.Int64(), t)
	if len(getRegimenPlanResponse.Sections) != 2 {
		t.Fatal("Expected 2 regimen sections")
	}

	//
	//
	// SUBMIT VISIT FOR PATIENT VISIT REVIEW
	//
	//

	// get doctor to submit the patient visit review
	SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	treatmentPlan, err = testData.DataAPI.GetAbridgedTreatmentPlan(treatmentPlan.ID.Int64(), doctor.DoctorID.Int64())
	test.OK(t, err)
	test.Equals(t, common.TPStatusSubmitted, treatmentPlan.Status)

	stubErxService.PrescriptionIdsToReturn = []int64{10, 20}
	stubErxService.PrescriptionIDToPrescriptionStatuses[10] = []common.StatusEvent{common.StatusEvent{Status: api.ERXStatusEntered}}
	stubErxService.PrescriptionIDToPrescriptionStatuses[20] = []common.StatusEvent{common.StatusEvent{Status: api.ERXStatusEntered}}
	doctor_treatment_plan.StartWorker(testData.DataAPI, stubErxService, testData.Config.Dispatcher, testData.Config.ERxRoutingQueue, testData.Config.ERxStatusQueue, 0, metrics.NewRegistry())

	// get an updated view of the patient informatio nfrom the database again given that weve assigned a prescription id to him
	patient, err = testData.DataAPI.GetPatientFromID(patient.PatientID.Int64())
	test.OK(t, err)

	prescriptionStatuses, err := testData.DataAPI.GetPrescriptionStatusEventsForPatient(patient.ERxPatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(prescriptionStatuses))

	for _, status := range prescriptionStatuses {
		test.Equals(t, api.ERXStatusSending, status.Status)
	}

	// at this point the closed date should be set on the visit
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(patientVisitResponse.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PVStatusTreated, patientVisit.Status)
	test.Equals(t, false, patientVisit.ClosedDate.IsZero())

	// attempt to consume the message put into the queue
	stubErxService.PrescriptionIDToPrescriptionStatuses[10] = []common.StatusEvent{common.StatusEvent{Status: api.ERXStatusSent}}
	stubErxService.PrescriptionIDToPrescriptionStatuses[20] = []common.StatusEvent{common.StatusEvent{Status: api.ERXStatusError, StatusDetails: "error test"}}

	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxService,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	statusWorker.Do()

	prescriptionStatuses, err = testData.DataAPI.GetPrescriptionStatusEventsForPatient(patient.ERxPatientID.Int64())
	test.OK(t, err)

	// there should be a total of 2 prescription statuses for this patient, with 1 per treatment
	test.Equals(t, 2, len(prescriptionStatuses))

	for _, status := range prescriptionStatuses {
		if status.ItemID == 20 && !(status.Status == api.ERXStatusError || status.Status == api.ERXStatusSending) {
			t.Fatal("Expected the prescription status to be error for 1 treatment")
		}

		if status.Status != api.ERXStatusSent && status.Status != api.ERXStatusSending && status.Status != api.ERXStatusError {
			t.Fatal("Expected the prescription status to be either eRxSent, Sending, or Error")
		}
	}

	treatments, err = testData.DataAPI.GetTreatmentsForPatient(patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(treatments))

	for _, treatment := range treatments {
		if treatment.ID.Int64() == 20 && (treatment.ERx.RxHistory[0].Status != api.ERXStatusError) {
			t.Fatal("Expected the prescription status to be error for 1 treatment")
		}

		if treatment.ERx.RxHistory[0].Status != api.ERXStatusSent && treatment.ERx.RxHistory[0].Status != api.ERXStatusSending && treatment.ERx.RxHistory[0].Status != api.ERXStatusError {
			t.Fatalf("Expected the prescription status to be either eRxSent, Sending, or Error. Instead it is %s", treatment.ERx.RxHistory[0].Status)
		}
	}

	//
	//
	// GET PATIENT VISIT REVIEW FOR PATIENT
	//
	//
	patient, err = testData.DataAPI.GetPatientFromID(patient.PatientID.Int64())
	test.OK(t, err)

	resp, err = testData.AuthGet(testData.APIServer.URL+apipaths.TreatmentPlanURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlan.ID.Int64(), 10), patient.AccountID.Int64())
	test.OK(t, err)
	resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}
