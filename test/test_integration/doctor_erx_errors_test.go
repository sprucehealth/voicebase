package test_integration

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/pharmacy"
)

func getTestRefillRequest(refillRequestQueueItemID, erxPatientID, prescriptionID, clinicianID, pharmacyID int64) *common.RefillRequestItem {
	return &common.RefillRequestItem{
		RxRequestQueueItemID:      refillRequestQueueItemID,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		RequestDateStamp:          time.Now(),
		ErxPatientID:              erxPatientID,
		PatientAddedForRequest:    false,
		ClinicianID:               clinicianID,
		RequestedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				erx.LexiDrugSynID:     "1234",
				erx.LexiGenProductID:  "12345",
				erx.LexiSynonymTypeID: "123556",
				erx.NDC:               "2415",
			},
			DosageStrength:       "10 mg",
			DispenseValue:        5,
			OTC:                  false,
			SubstitutionsAllowed: true,
			ERx: &common.ERxData{
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(prescriptionID),
				ErxPharmacyID:       pharmacyID,
			},
		},
		DispensedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				"drug_db_id_1": "1234",
				"drug_db_id_2": "12345",
			},
			DrugName:                "Teting (This - Drug)",
			DosageStrength:          "10 mg",
			DispenseValue:           5,
			DispenseUnitDescription: "Tablet",
			NumberRefills: encoding.NullInt64{
				IsValid:    true,
				Int64Value: 5,
			},
			SubstitutionsAllowed: false,
			DaysSupply: encoding.NullInt64{
				IsValid:    true,
				Int64Value: 10,
			}, PatientInstructions: "Take once daily",
			OTC: false,
			ERx: &common.ERxData{
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       pharmacyID,
				DoseSpotClinicianID: clinicianID,
			},
		},
	}
}

func getTestPreferredPharmacyAndTreatment() (*common.Treatment, *pharmacy.PharmacyData) {
	treatment1 := &common.Treatment{
		DrugDBIDs: map[string]string{
			erx.LexiDrugSynID:     "1234",
			erx.LexiGenProductID:  "12345",
			erx.LexiSynonymTypeID: "123556",
			erx.NDC:               "2415",
		},
		DrugInternalName:        "Drug1 (Route1 - Form1)",
		DosageStrength:          "Strength1",
		DispenseValue:           5,
		DispenseUnitDescription: "Tablet",
		DispenseUnitID:          encoding.NewObjectID(19),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		PatientInstructions: "Take once daily",
		OTC:                 false,
	}

	// create the preferred pharmacy for the patient
	pharmacySelection := &pharmacy.PharmacyData{
		SourceID:     12345,
		Source:       pharmacy.PharmacySourceSurescripts,
		AddressLine1: "12345 Marin Street",
		City:         "San Francisco",
		State:        "CA",
		Phone:        "12345667",
	}
	return treatment1, pharmacySelection
}

// Test treatment in treatment plan that has an error after being in the sent state
func TestRXError_Treatment_ErrorAfterSentState(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	// enable erx routing so that we can test the different expected status events
	// for the prescriptions
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	// setup test
	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id %s", err)
	}

	// get treatment ready for doctor to add for patient
	// while creating treatment plan
	prescriptionIDToReturn := int64(1235)
	treatment1, pharmacySelection := getTestPreferredPharmacyAndTreatment()

	// sign up a patient and get them to submit a patient visit
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	err = testData.DataAPI.UpdatePatientPharmacy(treatmentPlan.PatientID, pharmacySelection)
	if err != nil {
		t.Fatal("Unable to update patient pharmacy: " + err.Error())
	}

	treatmentResponse := AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)

	SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// ensure that the prescription is entered (rx started) so that it can be routed
	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PrescriptionIdsToReturn = []int64{prescriptionIDToReturn}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		prescriptionIDToReturn: []common.StatusEvent{common.StatusEvent{
			Status: api.ERXStatusEntered,
		},
		},
	}
	doctor_treatment_plan.StartWorker(testData.DataAPI, stubErxAPI, testData.Config.Dispatcher, testData.Config.ERxRoutingQueue, testData.Config.ERxStatusQueue, 0, metrics.NewRegistry())

	// once the treatment has been submitted, track the status of the submitted treatment to move it to the sent state
	stubErxAPI.PrescriptionIdsToReturn = []int64{prescriptionIDToReturn}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		prescriptionIDToReturn: []common.StatusEvent{common.StatusEvent{
			Status: api.ERXStatusSent,
		},
		},
	}

	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	statusWorker.Do()

	// expected state of the treatment here is sent
	statusEvents, err := testData.DataAPI.GetPrescriptionStatusEventsForTreatment(treatmentResponse.TreatmentList.Treatments[0].ID.Int64())
	if err != nil {
		t.Fatalf("Unable to get status events for treatments: %s", err)
	} else if len(statusEvents) != 2 {
		t.Fatalf("Expected 2 status events instead got %d", len(statusEvents))
	} else if statusEvents[0].Status != api.ERXStatusSent {
		t.Fatalf("Expected status to be %s instead it was %s", api.ERXStatusSent, statusEvents[0].Status)
	}

	// now stub the erx api to return a "free-standing" transmission error detail for this treatment
	stubErxAPI.TransmissionErrorsForPrescriptionIds = []int64{prescriptionIDToReturn}
	errorWorker := app_worker.NewERxErrorWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.MetricsRegistry)
	errorWorker.Do()

	// there should now be 3 status events for this treatment given that
	// the rx error checker caught the missed transition from sending -> sent -> error
	statusEvents, err = testData.DataAPI.GetPrescriptionStatusEventsForTreatment(treatmentResponse.TreatmentList.Treatments[0].ID.Int64())
	if err != nil {
		t.Fatalf("Unable to get status events for treatment: %s", err)
	} else if len(statusEvents) != 3 {
		t.Fatalf("Expected 3 status events instead got %d", len(statusEvents))
	} else if statusEvents[0].Status != api.ERXStatusError && statusEvents[1].Status != api.ERXStatusSent {
		t.Fatalf("Expected a transition from sent -> error, instead got %s -> %s", statusEvents[1].Status, statusEvents[0].Status)
	}

	// there should also be a pending item in the doctor's queue for the errored transmission
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor queue: %s", err)
	} else if len(pendingItems) != 1 {
		t.Fatalf("Expected 1 item in the pending tab of doctor queue instead got %d", len(pendingItems))
	} else if pendingItems[0].EventType != api.DQEventTypeTransmissionError {
		t.Fatalf("Expected a %s event type in the doctor queue instead got %s", api.DQEventTypeTransmissionError, pendingItems[0].EventType)
	}
}

// Test treatment in treatment plan that has an error after being in the sending state
func TestRXError_Treatment_ErrorAfterSendingState(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	// enable erx routing so that we can test the different expected status events
	// for the prescriptions
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	// setup test
	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id %s", err)
	}

	// get treatment ready for doctor to add for patient
	// while creating treatment plan
	prescriptionIDToReturn := int64(1235)
	treatment1, pharmacySelection := getTestPreferredPharmacyAndTreatment()

	// sign up a patient and get them to submit a patient visit
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	err = testData.DataAPI.UpdatePatientPharmacy(treatmentPlan.PatientID, pharmacySelection)
	if err != nil {
		t.Fatal("Unable to update patient pharmacy: " + err.Error())
	}

	// get the doctor to add a treatment to the patient visit that we can track the status of
	treatmentResponse := AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(),
		treatmentPlan.ID.Int64(), t)

	SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// first return the erx status as entered so that we can proceed forward with routing the erx
	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PrescriptionIdsToReturn = []int64{prescriptionIDToReturn}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		prescriptionIDToReturn: []common.StatusEvent{common.StatusEvent{
			Status: api.ERXStatusEntered,
		},
		},
	}

	doctor_treatment_plan.StartWorker(
		testData.DataAPI,
		testData.Config.ERxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxRoutingQueue,
		testData.Config.ERxStatusQueue,
		0,
		metrics.NewRegistry())

	stubErxAPI.PrescriptionIdsToReturn = []int64{prescriptionIDToReturn}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		prescriptionIDToReturn: []common.StatusEvent{common.StatusEvent{
			Status: api.ERXStatusSent,
		},
		},
	}

	// now stub the erx api to return a "free-standing" transmission error detail for this treatment
	stubErxAPI.TransmissionErrorsForPrescriptionIds = []int64{prescriptionIDToReturn}
	errorWorker := app_worker.NewERxErrorWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.MetricsRegistry)
	errorWorker.Do()

	// there should now be 2 status events for this treatment given that
	// the rx error checker caught the transition from sending  -> error
	statusEvents, err := testData.DataAPI.GetPrescriptionStatusEventsForTreatment(treatmentResponse.TreatmentList.Treatments[0].ID.Int64())
	if err != nil {
		t.Fatalf("Unable to get status events for treatment: %s", err)
	} else if len(statusEvents) != 2 {
		t.Fatalf("Expected 2 status events instead got %d", len(statusEvents))
	} else if statusEvents[0].Status != api.ERXStatusError && statusEvents[1].Status != api.ERXStatusSending {
		t.Fatalf("Expected a transition from sent -> error, instead got %s -> %s", statusEvents[1].Status, statusEvents[0].Status)
	}

	// there should also be a pending item in the doctor's queue for the errored transmission
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor queue: %s", err)
	} else if len(pendingItems) != 1 {
		t.Fatalf("Expected 1 item in the pending tab of doctor queue instead got %d", len(pendingItems))
	} else if pendingItems[0].EventType != api.DQEventTypeTransmissionError {
		t.Fatalf("Expected a %s event type in the doctor queue instead got %s", api.DQEventTypeTransmissionError, pendingItems[0].EventType)
	}
}

// Test treatment in treatment plan that has an error after being in the sent state
func TestRXError_Treatment_ErrorAfterError(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	// enable erx routing so that we can test the different expected status events
	// for the prescriptions
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	// setup test
	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id %s", err)
	}

	// get treatment ready for doctor to add for patient
	// while creating treatment plan
	prescriptionIDToReturn := int64(1235)
	treatment1, pharmacySelection := getTestPreferredPharmacyAndTreatment()

	// sign up a patient and get them to submit a patient visit
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	err = testData.DataAPI.UpdatePatientPharmacy(treatmentPlan.PatientID, pharmacySelection)
	if err != nil {
		t.Fatal("Unable to update patient pharmacy: " + err.Error())
	}

	// get the doctor to add a treatment to the patient visit that we can track the status of
	treatmentResponse := AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(),
		treatmentPlan.ID.Int64(), t)

	// first get the prescription status returned to be "Entered" so that it can be routed
	// by the worker
	SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PrescriptionIdsToReturn = []int64{prescriptionIDToReturn}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		prescriptionIDToReturn: []common.StatusEvent{common.StatusEvent{
			Status: api.ERXStatusEntered,
		},
		},
	}
	doctor_treatment_plan.StartWorker(testData.DataAPI, testData.Config.ERxAPI, testData.Config.Dispatcher, testData.Config.ERxRoutingQueue, testData.Config.ERxStatusQueue, 0, metrics.NewRegistry())

	stubErxAPI.PrescriptionIdsToReturn = []int64{prescriptionIDToReturn}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		prescriptionIDToReturn: []common.StatusEvent{common.StatusEvent{
			Status:        api.ERXStatusError,
			StatusDetails: "test error",
		},
		},
	}
	// once the treatment has been submitted, track the status of the submitted treatment to move it to the sent state
	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	statusWorker.Do()

	// expected state of the treatment here is sent
	statusEvents, err := testData.DataAPI.GetPrescriptionStatusEventsForTreatment(treatmentResponse.TreatmentList.Treatments[0].ID.Int64())
	if err != nil {
		t.Fatalf("Unable to get status events for treatments: %s", err)
	} else if len(statusEvents) != 2 {
		t.Fatalf("Expected 2 status events instead got %d", len(statusEvents))
	} else if statusEvents[0].Status != api.ERXStatusError {
		t.Fatalf("Expected status to be %s instead it was %s", api.ERXStatusSent, statusEvents[0].Status)
	}

	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor queue: %s", err)
	} else if len(pendingItems) != 1 {
		t.Fatalf("Expected 1 item in the pending tab of doctor queue instead got %d", len(pendingItems))
	} else if pendingItems[0].EventType != api.DQEventTypeTransmissionError {
		t.Fatalf("Expected a %s event type in the doctor queue instead got %s", api.DQEventTypeTransmissionError, pendingItems[0].EventType)
	}

	// now stub the erx api to return a "free-standing" transmission error detail for this treatment
	stubErxAPI.TransmissionErrorsForPrescriptionIds = []int64{prescriptionIDToReturn}
	errorWorker := app_worker.NewERxErrorWorker(
		testData.DataAPI,
		testData.ERxAPI,
		&TestLock{},
		testData.Config.MetricsRegistry)
	errorWorker.Do()

	// there should now be 3 status events for this treatment given that
	// the rx error checker caught the missed transition from sending -> sent -> error
	statusEvents, err = testData.DataAPI.GetPrescriptionStatusEventsForTreatment(treatmentResponse.TreatmentList.Treatments[0].ID.Int64())
	if err != nil {
		t.Fatalf("Unable to get status events for treatment: %s", err)
	} else if len(statusEvents) != 2 {
		t.Fatalf("Expected 3 status events instead got %d", len(statusEvents))
	} else if statusEvents[0].Status != api.ERXStatusError && statusEvents[1].Status != api.ERXStatusError {
		t.Fatalf("Expected a transition from sent -> error, instead got %s -> %s", statusEvents[1].Status, statusEvents[0].Status)
	}

	// there should also be a pending item in the doctor's queue for the errored transmission
	pendingItems, err = testData.DataAPI.GetPendingItemsInDoctorQueue(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor queue: %s", err)
	} else if len(pendingItems) != 1 {
		t.Fatalf("Expected 1 item in the pending tab of doctor queue instead got %d", len(pendingItems))
	} else if pendingItems[0].EventType != api.DQEventTypeTransmissionError {
		t.Fatalf("Expected a %s event type in the doctor queue instead got %s", api.DQEventTypeTransmissionError, pendingItems[0].EventType)
	}
}

func TestRXError_Refill_ErrorAfterSentState(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	// enable erx routing so that we can test the different expected status events
	// for the prescriptions
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	doctor := createDoctorWithClinicianID(testData, t)
	erxPatientID := int64(123556)
	pharmacyID := int64(12345)
	prescriptionIDForRequestedPrescription := int64(12314)
	refillRequestQueueItemID := int64(12421415)
	approvedRefillRequestPrescriptionID := int64(124424242)

	// add pharmacy to database so that it can be linked to treatment that is added
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     pharmacyID,
		Source:       pharmacy.PharmacySourceSurescripts,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}
	if err := testData.DataAPI.AddPharmacy(pharmacyToReturn); err != nil {
		t.Fatalf("Unable to add pharmacy to database: %s", err)
	}

	// get stub erx api to return pharmacy details
	patientToReturn := &common.Patient{
		FirstName:    "Test",
		LastName:     "TestLastName",
		DOB:          encoding.Date{Year: 1987, Month: 1, Day: 22},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(erxPatientID),
	}

	refillRequestItem := getTestRefillRequest(refillRequestQueueItemID, erxPatientID, prescriptionIDForRequestedPrescription, doctor.DoseSpotClinicianID, pharmacyID)

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}
	stubErxAPI.RefillRequestPrescriptionIds = map[int64]int64{
		refillRequestQueueItemID: approvedRefillRequestPrescriptionID,
	}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		approvedRefillRequestPrescriptionID: []common.StatusEvent{common.StatusEvent{
			Status: api.ERXStatusSent,
		},
		},
	}

	// consume the refill request to store the refill request into our system
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	refillRXWorker.Do()

	// now lets go ahead and attempt to approve this refill request
	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to get pending refill requests from clinic: " + err.Error())
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatalf("Unable to get refill request from database: %s", err)
	}

	approveRefillRequest(refillRequest, doctor.AccountID.Int64(), "this is a test", testData, t)

	// now that the refill request has been approved there should be an item in the message queue to check the status of the
	// prescription that was created as a result of the approval. Let's get this prescription to transition from approved -> sent
	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	statusWorker.Do()

	// now lets get it to transition into the ERROR state
	stubErxAPI.TransmissionErrorsForPrescriptionIds = []int64{approvedRefillRequestPrescriptionID}
	errorWorker := app_worker.NewERxErrorWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.MetricsRegistry)
	errorWorker.Do()

	refillStatusEvents, err := testData.DataAPI.GetRefillStatusEventsForRefillRequest(refillRequest.ID)
	if err != nil {
		t.Fatalf("Unable to get refill status events: %s", err)
	} else if len(refillStatusEvents) != 4 {
		t.Fatalf("Expected 4 refill status events instead got %d", len(refillStatusEvents))
	} else if refillStatusEvents[0].Status != api.RXRefillStatusError && refillStatusEvents[1].Status != api.RXRefillStatusSent {
		t.Fatalf("Expected the refill request prescription to transition from SENT -> ERROR but instead it was %s -> %s ", refillRequestStatuses[1].Status, refillRequestStatuses[0].Status)
	}
}

func TestRXError_Refill_ErrorAfterSendingState(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	// enable erx routing so that we can test the different expected status events
	// for the prescriptions
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	doctor := createDoctorWithClinicianID(testData, t)
	// patientResponse := SignupRandomTestPatientWithPharmacyAndAddress(t, testData.DataApi, testData.AuthApi)
	erxPatientID := int64(123556)
	pharmacyID := int64(12345)
	prescriptionIDForRequestedPrescription := int64(12314)
	refillRequestQueueItemID := int64(12421415)
	approvedRefillRequestPrescriptionID := int64(124424242)

	// add pharmacy to database so that it can be linked to treatment that is added
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     pharmacyID,
		Source:       pharmacy.PharmacySourceSurescripts,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}
	if err := testData.DataAPI.AddPharmacy(pharmacyToReturn); err != nil {
		t.Fatalf("Unable to add pharmacy to database: %s", err)
	}

	// get stub erx api to return pharmacy details
	patientToReturn := &common.Patient{
		FirstName:    "Test",
		LastName:     "TestLastName",
		DOB:          encoding.Date{Year: 1987, Month: 1, Day: 22},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(erxPatientID),
	}

	refillRequestItem := getTestRefillRequest(refillRequestQueueItemID, erxPatientID, prescriptionIDForRequestedPrescription, doctor.DoseSpotClinicianID, pharmacyID)
	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}
	stubErxAPI.RefillRequestPrescriptionIds = map[int64]int64{
		refillRequestQueueItemID: approvedRefillRequestPrescriptionID,
	}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		approvedRefillRequestPrescriptionID: []common.StatusEvent{common.StatusEvent{
			Status: api.ERXStatusSent,
		},
		},
	}

	// consume the refill request to store the refill request into our system
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	refillRXWorker.Do()

	// now lets go ahead and attempt to approve this refill request
	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to get pending refill requests from clinic: " + err.Error())
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatalf("Unable to get refill request from database: %s", err)
	}

	approveRefillRequest(refillRequest, doctor.AccountID.Int64(), "this is a test", testData, t)

	// now lets get it to transition into the ERROR state
	stubErxAPI.TransmissionErrorsForPrescriptionIds = []int64{approvedRefillRequestPrescriptionID}
	errorWorker := app_worker.NewERxErrorWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.MetricsRegistry)
	errorWorker.Do()

	refillStatusEvents, err := testData.DataAPI.GetRefillStatusEventsForRefillRequest(refillRequest.ID)
	if err != nil {
		t.Fatalf("Unable to get refill status events: %s", err)
	} else if len(refillStatusEvents) != 3 {
		t.Fatalf("Expected 3 refill status events instead got %d", len(refillStatusEvents))
	} else if refillStatusEvents[0].Status != api.RXRefillStatusError && refillStatusEvents[1].Status != api.RXRefillStatusApproved {
		t.Fatalf("Expected the refill request prescription to transition from SENT -> ERROR but instead it was %s -> %s ", refillRequestStatuses[1].Status, refillRequestStatuses[0].Status)
	}
}

func TestRXError_Refill_ErrorAfterErrorState(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	// enable erx routing so that we can test the different expected status events
	// for the prescriptions
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	doctor := createDoctorWithClinicianID(testData, t)
	// patientResponse := SignupRandomTestPatientWithPharmacyAndAddress(t, testData.DataApi, testData.AuthApi)
	erxPatientID := int64(123556)
	pharmacyID := int64(12345)
	prescriptionIDForRequestedPrescription := int64(12314)
	refillRequestQueueItemID := int64(12421415)
	approvedRefillRequestPrescriptionID := int64(124424242)

	// add pharmacy to database so that it can be linked to treatment that is added
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     pharmacyID,
		Source:       pharmacy.PharmacySourceSurescripts,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}
	if err := testData.DataAPI.AddPharmacy(pharmacyToReturn); err != nil {
		t.Fatalf("Unable to add pharmacy to database: %s", err)
	}

	// get stub erx api to return pharmacy details
	patientToReturn := &common.Patient{
		FirstName:    "Test",
		LastName:     "TestLastName",
		DOB:          encoding.Date{Year: 1987, Month: 1, Day: 22},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(erxPatientID),
	}

	refillRequestItem := getTestRefillRequest(refillRequestQueueItemID, erxPatientID, prescriptionIDForRequestedPrescription, doctor.DoseSpotClinicianID, pharmacyID)
	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}
	stubErxAPI.RefillRequestPrescriptionIds = map[int64]int64{
		refillRequestQueueItemID: approvedRefillRequestPrescriptionID,
	}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		approvedRefillRequestPrescriptionID: []common.StatusEvent{common.StatusEvent{
			Status:        api.ERXStatusError,
			StatusDetails: "Error state",
		},
		},
	}

	// consume the refill request to store the refill request into our system
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	refillRXWorker.Do()

	// now lets go ahead and attempt to approve this refill request
	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to get pending refill requests from clinic: " + err.Error())
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatalf("Unable to get refill request from database: %s", err)
	}

	approveRefillRequest(refillRequest, doctor.AccountID.Int64(), "this is a test", testData, t)

	// now that the refill request has been approved there should be an item in the message queue to check the status of the
	// prescription that was created as a result of the approval. Let's get this prescription to transition from approved -> sent
	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	statusWorker.Do()

	// now lets get it to transition into the ERROR state
	stubErxAPI.TransmissionErrorsForPrescriptionIds = []int64{approvedRefillRequestPrescriptionID}
	statusWorker.Do()

	refillStatusEvents, err := testData.DataAPI.GetRefillStatusEventsForRefillRequest(refillRequest.ID)
	if err != nil {
		t.Fatalf("Unable to get refill status events: %s", err)
	} else if len(refillStatusEvents) != 3 {
		t.Fatalf("Expected 3 refill status events instead got %d", len(refillStatusEvents))
	} else if refillStatusEvents[0].Status != api.RXRefillStatusError && refillStatusEvents[1].Status != api.RXRefillStatusApproved {
		t.Fatalf("Expected the refill request prescription to transition from APPROVED -> ERROR but instead it was %s -> %s ", refillRequestStatuses[1].Status, refillRequestStatuses[0].Status)
	}
}

// Test unlinked dntf treatment that has an error after being in sent state
func TestRXError_UnlinkedDNTFT_SentToErrorState(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	// enable erx routing so that we can test the different expected status events
	// for the prescriptions
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	unlinkedTreatment := setupRefill_Deny_DNTF(t, testData, common.StatusEvent{Status: api.ERXStatusSent}, false)

	// lets go ahead and setup the stubErxApi to throw a transmission error for this treatment now
	stubErxAPI := &erx.StubErxService{
		TransmissionErrorsForPrescriptionIds: []int64{unlinkedTreatment.ERx.PrescriptionID.Int64()},
	}
	errorWorker := app_worker.NewERxErrorWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.MetricsRegistry)
	errorWorker.Do()

	unlinkedTreatment, err := testData.DataAPI.GetUnlinkedDNTFTreatment(unlinkedTreatment.ID.Int64())
	if err != nil {
		t.Fatalf(err.Error())
	} else if len(unlinkedTreatment.ERx.RxHistory) != 4 {
		t.Fatalf("Expected 4 items in the rx history of an unlinked dntf treatment")
	} else if unlinkedTreatment.ERx.RxHistory[0].Status != api.ERXStatusError || unlinkedTreatment.ERx.RxHistory[1].Status != api.ERXStatusSent {
		t.Fatalf("Expected rx history to go from Sent -> Error instead it was frmo %s -> %s", unlinkedTreatment.ERx.RxHistory[1].Status, unlinkedTreatment.ERx.RxHistory[0].Status)
	}
}

func TestRXError_UnlinkedDNTF_SendingToErrorState(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	// enable erx routing so that we can test the different expected status events
	// for the prescriptions
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	unlinkedTreatment := setupRefill_Deny_DNTF(t, testData, common.StatusEvent{Status: api.ERXStatusError}, false)

	// lets go ahead and setup the stubErxApi to throw a transmission error for this treatment now
	stubErxAPI := &erx.StubErxService{
		TransmissionErrorsForPrescriptionIds: []int64{unlinkedTreatment.ERx.PrescriptionID.Int64()},
	}
	errorWorker := app_worker.NewERxErrorWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.MetricsRegistry)
	errorWorker.Do()

	unlinkedTreatment, err := testData.DataAPI.GetUnlinkedDNTFTreatment(unlinkedTreatment.ID.Int64())
	if err != nil {
		t.Fatalf(err.Error())
	} else if len(unlinkedTreatment.ERx.RxHistory) != 3 {
		t.Fatalf("Expected 4 items in the rx history of an unlinked dntf treatment")
	} else if unlinkedTreatment.ERx.RxHistory[0].Status != api.ERXStatusError || unlinkedTreatment.ERx.RxHistory[1].Status != api.ERXStatusSending {
		t.Fatalf("Expected rx history to go from Sending -> Error instead it was from %s -> %s", unlinkedTreatment.ERx.RxHistory[1].Status, unlinkedTreatment.ERx.RxHistory[0].Status)
	}
}
