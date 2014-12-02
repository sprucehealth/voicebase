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

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from id: " + err.Error())
	}

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf("Unable to get patient from patient visit info: %s", err)
	}

	pharmacySelection := &pharmacy.PharmacyData{
		SourceId:     12345,
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

	resp, err := testData.AuthGet(testData.APIServer.URL+apipaths.TreatmentPlanURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlan.Id.Int64(), 10), patient.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusNotFound, resp.StatusCode)

	// once the doctor has started reviewing the case, lets go ahead and get the doctor to close the case with no diagnosis
	stubErxService := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxService.PatientErxId = 10
	stubErxService.PrescriptionIdsToReturn = []int64{}
	stubErxService.PrescriptionIdToPrescriptionStatuses = make(map[int64][]common.StatusEvent)
	stubErxService.PharmacyToSendPrescriptionTo = pharmacySelection.SourceId

	SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)
	// consume the message
	doctor_treatment_plan.StartWorker(testData.DataApi, stubErxService, testData.Config.Dispatcher, testData.Config.ERxRoutingQueue, testData.Config.ERxStatusQueue, 0, metrics.NewRegistry())

	// start a new patient visit
	patientVisitResponse, treatmentPlan = CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err = testData.DataApi.GetPatientFromPatientVisitId(patientVisitResponse.PatientVisitId)
	test.OK(t, err)

	err = testData.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), pharmacySelection)
	test.OK(t, err)

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

	getTreatmentsResponse := AddAndGetTreatmentsForPatientVisit(testData, treatments, doctor.AccountId.Int64(), treatmentPlan.Id.Int64(), t)
	if len(getTreatmentsResponse.TreatmentList.Treatments) != 2 {
		t.Fatalf("Expected 2 treatments to be returned, instead got back %d", len(getTreatmentsResponse.TreatmentList.Treatments))
	}

	//
	//
	// SUBMIT REGIMEN PLAN
	//
	//
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: treatmentPlan.Id,
	}

	regimenStep1 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 1",
		State: common.STATE_ADDED,
	}

	regimenStep2 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 2",
		State: common.STATE_ADDED,
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
	getRegimenPlanResponse := GetRegimenPlanForTreatmentPlan(testData, doctor, treatmentPlan.Id.Int64(), t)
	if len(getRegimenPlanResponse.Sections) != 2 {
		t.Fatal("Expected 2 regimen sections")
	}

	//
	//
	// SUBMIT VISIT FOR PATIENT VISIT REVIEW
	//
	//

	// get doctor to submit the patient visit review
	SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	treatmentPlan, err = testData.DataApi.GetAbridgedTreatmentPlan(treatmentPlan.Id.Int64(), doctor.DoctorId.Int64())
	test.OK(t, err)
	test.Equals(t, common.TPStatusSubmitted, treatmentPlan.Status)

	stubErxService.PrescriptionIdsToReturn = []int64{10, 20}
	stubErxService.PrescriptionIdToPrescriptionStatuses[10] = []common.StatusEvent{common.StatusEvent{Status: api.ERX_STATUS_ENTERED}}
	stubErxService.PrescriptionIdToPrescriptionStatuses[20] = []common.StatusEvent{common.StatusEvent{Status: api.ERX_STATUS_ENTERED}}
	doctor_treatment_plan.StartWorker(testData.DataApi, stubErxService, testData.Config.Dispatcher, testData.Config.ERxRoutingQueue, testData.Config.ERxStatusQueue, 0, metrics.NewRegistry())

	// get an updated view of the patient informatio nfrom the database again given that weve assigned a prescription id to him
	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	test.OK(t, err)

	prescriptionStatuses, err := testData.DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(prescriptionStatuses))

	for _, status := range prescriptionStatuses {
		test.Equals(t, api.ERX_STATUS_SENDING, status.Status)
	}

	// at this point the closed date should be set on the visit
	patientVisit, err := testData.DataApi.GetPatientVisitFromId(patientVisitResponse.PatientVisitId)
	test.OK(t, err)
	test.Equals(t, common.PVStatusTreated, patientVisit.Status)
	test.Equals(t, false, patientVisit.ClosedDate.IsZero())

	// attempt to consume the message put into the queue
	stubErxService.PrescriptionIdToPrescriptionStatuses[10] = []common.StatusEvent{common.StatusEvent{Status: api.ERX_STATUS_SENT}}
	stubErxService.PrescriptionIdToPrescriptionStatuses[20] = []common.StatusEvent{common.StatusEvent{Status: api.ERX_STATUS_ERROR, StatusDetails: "error test"}}
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxService, testData.Config.Dispatcher, testData.Config.ERxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	prescriptionStatuses, err = testData.DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId.Int64())
	test.OK(t, err)

	// there should be a total of 2 prescription statuses for this patient, with 1 per treatment
	test.Equals(t, 2, len(prescriptionStatuses))

	for _, status := range prescriptionStatuses {
		if status.ItemId == 20 && !(status.Status == api.ERX_STATUS_ERROR || status.Status == api.ERX_STATUS_SENDING) {
			t.Fatal("Expected the prescription status to be error for 1 treatment")
		}

		if status.Status != api.ERX_STATUS_SENT && status.Status != api.ERX_STATUS_SENDING && status.Status != api.ERX_STATUS_ERROR {
			t.Fatal("Expected the prescription status to be either eRxSent, Sending, or Error")
		}
	}

	treatments, err = testData.DataApi.GetTreatmentsForPatient(patient.PatientId.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(treatments))

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
	test.OK(t, err)

	resp, err = testData.AuthGet(testData.APIServer.URL+apipaths.TreatmentPlanURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlan.Id.Int64(), 10), patient.AccountId.Int64())
	test.OK(t, err)
	resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}
