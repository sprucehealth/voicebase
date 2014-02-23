package integration

import (
	"carefront/api"
	"carefront/app_worker"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"testing"
	"time"

	"github.com/samuel/go-metrics/metrics"
)

const (
	clinicianId = 100
)

func TestNewRefillRequestForExistingPatientAndExistingTreatment(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianId(testData, t)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	erxPatientId := int64(60)

	// add an erx patient id to the patient
	err := testData.DataApi.UpdatePatientWithERxPatientId(signedupPatientResponse.Patient.PatientId.Int64(), erxPatientId)
	if err != nil {
		t.Fatal("Unable to update patient with erx patient id : " + err.Error())
	}

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceId:     "1234",
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err = testData.DataApi.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	patientVisitResponse := CreatePatientVisitForPatient(signedupPatientResponse.Patient.PatientId.Int64(), testData, t)
	// start a new treatemtn plan for the patient visit
	treatmentPlanId, err := testData.DataApi.StartNewTreatmentPlanForPatientVisit(signedupPatientResponse.Patient.PatientId.Int64(),
		patientVisitResponse.PatientVisitId, doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to start new treatment plan for patient visit " + err.Error())
	}

	testTime := time.Now()

	treatment1 := &common.Treatment{
		PrescriptionId:     common.NewObjectId(5504),
		PrescriptionStatus: "Requested",
		ErxPharmacyId:      1234,
		PharmacyLocalId:    common.NewObjectId(pharmacyToReturn.LocalId),
		DrugDBIds: map[string]string{
			"drug_db_id_1": "1234",
			"drug_db_id_2": "12345",
		},
		DrugName:                "Teting (This - Drug)",
		DosageStrength:          "10 mg",
		DispenseValue:           5,
		DispenseUnitDescription: "Tablet",
		DispenseUnitId:          common.NewObjectId(19),
		NumberRefills:           5,
		SubstitutionsAllowed:    false,
		DaysSupply:              10,
		PatientInstructions:     "Take once daily",
		ErxLastDateFilled:       &testTime,
		OTC:                     false,
	}

	// add this treatment to the treatment plan
	err = testData.DataApi.AddTreatmentsForPatientVisit([]*common.Treatment{treatment1}, doctor.DoctorId.Int64(), treatmentPlanId)
	if err != nil {
		t.Fatal("Unable to add treatment for patient visit: " + err.Error())
	}

	// update the treatment with prescription id and pharmacy id for where prescription was routed
	_, err = testData.DB.Exec(`update treatment set erx_id = ?, pharmacy_id=? where id = ?`, treatment1.PrescriptionId.Int64(), pharmacyToReturn.LocalId, treatment1.Id.Int64())
	if err != nil {
		t.Fatal("Unable to update treatment with erx id: " + err.Error())
	}

	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemId:      12345,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		RequestedDrugDescription:  "Clyndamycin",
		RequestedRefillAmount:     "10",
		RequestedDispense:         "123",
		ErxPatientId:              erxPatientId,
		PatientAddedForRequest:    false,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription:     treatment1,
		DispensedPrescription: &common.Treatment{
			PrescriptionId:     common.NewObjectId(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      1234,
			DrugDBIds: map[string]string{
				"drug_db_id_1": "1234",
				"drug_db_id_2": "12345",
			},
			DrugName:                "Teting (This - Drug)",
			DosageStrength:          "10 mg",
			DispenseValue:           5,
			DispenseUnitDescription: "Tablet",
			NumberRefills:           5,
			SubstitutionsAllowed:    false,
			DaysSupply:              10,
			PatientInstructions:     "Take once daily",
			ErxLastDateFilled:       &testTime,
			OTC:                     false,
		},
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter())

	// There should be an unlinked treatment in the unlinked_requested_treatment db
	var count int64
	err = testData.DB.QueryRow(`select count(*) from unlinked_requested_treatment`).Scan(&count)
	if err != nil {
		t.Fatal("Unable to get a count for the unumber of treatments in the unlinked_requested_treatment table " + err.Error())
	}
	if count != 0 {
		t.Fatalf("Expected there to be no unlinked treatment, but got %d unlinked treatments instead", count)
	}

	// There should be a status entry in the refill_request_status table
	refillRequestStatuses, err := testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	if len(refillRequestStatuses) != 1 {
		t.Fatal("Expected there to exist 1 refill request status for the refill request just persisted")
	}

	if refillRequestStatuses[0].RxRequestQueueItemId != refillRequestItem.RxRequestQueueItemId ||
		refillRequestStatuses[0].Status != api.RX_REFILL_STATUS_REQUESTED {
		t.Fatal("Refill request status not in expected state")
	}

	// There should be a pending entry in the doctor's queue
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get pending items from doctor queue: " + err.Error())
	}

	if len(pendingItems) != 1 {
		t.Fatal("Expected there to exist 1 pending item in the doctor's queue which is the refill request")
	}

	if pendingItems[0].EventType != api.EVENT_TYPE_REFILL_REQUEST ||
		pendingItems[0].ItemId != refillRequestStatuses[0].ErxRefillRequestId {
		t.Fatal("Pending item found in the doctor's queue is not the expected item")
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ErxRefillRequestId)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	if refillRequest.DispensedPrescription == nil {
		t.Fatalf("Dispensed prescription was null for the refill request when it shouldn't be")
	}

	if refillRequest.RequestedPrescription == nil {
		t.Fatal("Requested prescription was null for refill request when it shouldn't be")
	}

	if refillRequest.RequestedPrescription.IsUnlinked {
		t.Fatal("Requested prescription should be one that was found in our system, but instead its indicated to be unlinked")
	}

	if refillRequest.RequestedPrescription.TreatmentPlanId == nil || refillRequest.RequestedPrescription.TreatmentPlanId.Int64() == 0 {
		t.Fatal("Requested prescription expected to have a treatment plan id set given that it was found linked to one of the treatments in our system")
	}

	if refillRequest.RequestedPrescription.PatientVisitId == nil || refillRequest.RequestedPrescription.PatientVisitId.Int64() == 0 {
		t.Fatal("Requested prescription expected to have a patient visit id set given that it was found linked to one of the treatments in our system")
	}

	if refillRequest.Patient == nil {
		t.Fatal("Refill request expected to have patient demographics attached to it instead it doesnt")
	}

	if refillRequest.Patient.Status != api.PATIENT_REGISTERED {
		t.Fatal("Patient requesting refill expected to be in our system instead the indication is that it was an unlinked patient")
	}
}

func TestNewRefillRequestWithUnlinkedTreatmentAndLinkedPatient(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianId(testData, t)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	erxPatientId := int64(60)

	// add an erx patient id to the patient
	err := testData.DataApi.UpdatePatientWithERxPatientId(signedupPatientResponse.Patient.PatientId.Int64(), erxPatientId)
	if err != nil {
		t.Fatal("Unable to update patient with erx patient id : " + err.Error())
	}

	testTime := time.Now()
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemId:      12345,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		RequestedDrugDescription:  "Clyndamycin",
		RequestedRefillAmount:     "10",
		RequestedDispense:         "123",
		ErxPatientId:              erxPatientId,
		PatientAddedForRequest:    false,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
			PrescriptionId:     common.NewObjectId(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      123,
			DrugDBIds: map[string]string{
				"drug_db_id_1": "1234",
				"drug_db_id_2": "12345",
			},
			DrugName:                "Teting (This - Drug)",
			DosageStrength:          "10 mg",
			DispenseValue:           5,
			DispenseUnitDescription: "Tablet",
			NumberRefills:           5,
			SubstitutionsAllowed:    false,
			DaysSupply:              10,
			PatientInstructions:     "Take once daily",
			ErxLastDateFilled:       &testTime,
			OTC:                     false,
		},
		DispensedPrescription: &common.Treatment{
			PrescriptionId:     common.NewObjectId(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      123,
			DrugDBIds: map[string]string{
				"drug_db_id_1": "1234",
				"drug_db_id_2": "12345",
			},
			DrugName:                "Teting (This - Drug)",
			DosageStrength:          "10 mg",
			DispenseValue:           5,
			DispenseUnitDescription: "Tablet",
			NumberRefills:           5,
			SubstitutionsAllowed:    false,
			DaysSupply:              10,
			PatientInstructions:     "Take once daily",
			ErxLastDateFilled:       &testTime,
			OTC:                     false,
		},
	}

	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceId:     "1234",
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter())

	// TODO Get Refill Request when that API is written, and ensure that the state of the refill request matches the
	// end expected state (patient that is unlinked; treatment that is unlinked; pharmacy data in there)

	// There should be an unlinked patient in the patient db
	linkedpatient, err := testData.DataApi.GetPatientFromErxPatientId(erxPatientId)
	if err != nil {
		t.Fatal("Unable to get patient based on erx patient id to verify the patient information: " + err.Error())
	}

	if linkedpatient.Status != api.PATIENT_REGISTERED {
		t.Fatal("Patient was expected to be registered but it was not")
	}

	// There should be an unlinked pharmacy treatment in the unlinked_requested_treatment db
	// There should be a dispensed treatment in the pharmacy_dispensed_treatment db
	// There should be a test pharmacy in the pharmacy_selection db
	// There should be a status entry in the refill_request_status table
	refillRequestStatuses, err := testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	if len(refillRequestStatuses) != 1 {
		t.Fatal("Expected there to exist 1 refill request status for the refill request just persisted")
	}

	if refillRequestStatuses[0].RxRequestQueueItemId != refillRequestItem.RxRequestQueueItemId ||
		refillRequestStatuses[0].Status != api.RX_REFILL_STATUS_REQUESTED {
		t.Fatal("Refill request status not in expected state")
	}

	// There should be a pending entry in the doctor's queue
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get pending items from doctor queue: " + err.Error())
	}

	if len(pendingItems) != 1 {
		t.Fatal("Expected there to exist 1 pending item in the doctor's queue which is the refill request")
	}

	if pendingItems[0].EventType != api.EVENT_TYPE_REFILL_REQUEST ||
		pendingItems[0].ItemId != refillRequestStatuses[0].ErxRefillRequestId {
		t.Fatal("Pending item found in the doctor's queue is not the expected item")
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ErxRefillRequestId)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error)
	}

	if refillRequest.DispensedPrescription == nil {
		t.Fatalf("Dispensed prescription was null for the refill request when it shouldn't be")
	}

	if refillRequest.RequestedPrescription == nil {
		t.Fatal("Requested prescription was null for refill request when it shouldn't be")
	}

	if !refillRequest.RequestedPrescription.IsUnlinked {
		t.Fatal("Requested prescription should be unlinked but was instead found in the system")
	}

	if refillRequest.RequestedPrescription.TreatmentPlanId != nil || refillRequest.RequestedPrescription.TreatmentPlanId.Int64() != 0 {
		t.Fatal("Requested prescription not expected to have treatment plan id given that it was unlinked, instead it does")
	}

	if refillRequest.RequestedPrescription.PatientVisitId != nil || refillRequest.RequestedPrescription.PatientVisitId.Int64() != 0 {
		t.Fatal("Requested prescription not expected to have patient visit id given that it was unlinked, instead it does")
	}

	if refillRequest.Patient == nil {
		t.Fatal("Refill request expected to have patient demographics attached to it instead it doesnt")
	}

	if refillRequest.Patient.Status != api.PATIENT_REGISTERED {
		t.Fatal("Patient requesting refill expected to be in our system instead the indication is that it was an unlinked patient")
	}
}

func TestNewRefillRequestWithUnlinkedTreatmentAndUnlinkedPatient(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianId(testData, t)

	testTime := time.Now()
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemId:      12345,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		RequestedDrugDescription:  "Clyndamycin",
		RequestedRefillAmount:     "10",
		RequestedDispense:         "123",
		ErxPatientId:              555,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
			PrescriptionId:     common.NewObjectId(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      123,
			DrugDBIds: map[string]string{
				"drug_db_id_1": "1234",
				"drug_db_id_2": "12345",
			},
			DrugName:                "Teting (This - Drug)",
			DosageStrength:          "10 mg",
			DispenseValue:           5,
			DispenseUnitDescription: "Tablet",
			NumberRefills:           5,
			SubstitutionsAllowed:    false,
			DaysSupply:              10,
			PatientInstructions:     "Take once daily",
			ErxLastDateFilled:       &testTime,
			OTC:                     false,
		},
		DispensedPrescription: &common.Treatment{
			PrescriptionId:     common.NewObjectId(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      123,
			DrugDBIds: map[string]string{
				"drug_db_id_1": "1234",
				"drug_db_id_2": "12345",
			},
			DrugName:                "Teting (This - Drug)",
			DosageStrength:          "10 mg",
			DispenseValue:           5,
			DispenseUnitDescription: "Tablet",
			NumberRefills:           5,
			SubstitutionsAllowed:    false,
			DaysSupply:              10,
			PatientInstructions:     "Take once daily",
			ErxLastDateFilled:       &testTime,
			OTC:                     false,
		},
	}

	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceId:     "1234",
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	// Get StubErx to return patient details in the GetPatientDetails call
	patientToReturn := &common.Patient{
		FirstName:    "Test",
		LastName:     "TestLastName",
		Dob:          time.Now(),
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		City:         "Beverly Hills",
		State:        "CA",
		ERxPatientId: common.NewObjectId(12345),
	}

	stubErxAPI := &erx.StubErxService{
		PatientDetailsToReturn:       patientToReturn,
		PharmacyDetailsToReturn:      pharmacyToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter())

	// TODO Get Refill Request when that API is written, and ensure that the state of the refill request matches the
	// end expected state (patient that is unlinked; treatment that is unlinked; pharmacy data in there)

	// There should be an unlinked patient in the patient db
	unlinkedPatient, err := testData.DataApi.GetPatientFromErxPatientId(patientToReturn.ERxPatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient based on erx patient id to verify the patient information: " + err.Error())
	}

	if unlinkedPatient.Status != api.PATIENT_UNLINKED {
		t.Fatal("Patient was expected to be unlinked but it was not")
	}

	// There should be an unlinked pharmacy treatment in the unlinked_requested_treatment db
	// There should be a dispensed treatment in the pharmacy_dispensed_treatment db
	// There should be a test pharmacy in the pharmacy_selection db

	// There should be a status entry in the refill_request_status table
	refillRequestStatuses, err := testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	if len(refillRequestStatuses) != 1 {
		t.Fatal("Expected there to exist 1 refill request status for the refill request just persisted")
	}

	if refillRequestStatuses[0].RxRequestQueueItemId != refillRequestItem.RxRequestQueueItemId ||
		refillRequestStatuses[0].Status != api.RX_REFILL_STATUS_REQUESTED {
		t.Fatal("Refill request status not in expected state")
	}

	// There should be a pending entry in the doctor's queue
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get pending items from doctor queue: " + err.Error())
	}

	if len(pendingItems) != 1 {
		t.Fatal("Expected there to exist 1 pending item in the doctor's queue which is the refill request")
	}

	if pendingItems[0].EventType != api.EVENT_TYPE_REFILL_REQUEST ||
		pendingItems[0].ItemId != refillRequestStatuses[0].ErxRefillRequestId {
		t.Fatal("Pending item found in the doctor's queue is not the expected item")
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ErxRefillRequestId)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error)
	}

	if refillRequest.DispensedPrescription == nil {
		t.Fatalf("Dispensed prescription was null for the refill request when it shouldn't be")
	}

	if refillRequest.RequestedPrescription == nil {
		t.Fatal("Requested prescription was null for refill request when it shouldn't be")
	}

	if !refillRequest.RequestedPrescription.IsUnlinked {
		t.Fatal("Requested prescription should be unlinked but was instead found in the system")
	}

	if refillRequest.RequestedPrescription.TreatmentPlanId != nil || refillRequest.RequestedPrescription.TreatmentPlanId.Int64() != 0 {
		t.Fatal("Requested prescription not expected to have treatment plan id given that it was unlinked, instead it does")
	}

	if refillRequest.RequestedPrescription.PatientVisitId != nil || refillRequest.RequestedPrescription.PatientVisitId.Int64() != 0 {
		t.Fatal("Requested prescription not expected to have patient visit id given that it was unlinked, instead it does")
	}

	if refillRequest.Patient == nil {
		t.Fatal("Refill request expected to have patient demographics attached to it instead it doesnt")
	}

	if refillRequest.Patient.Status != api.PATIENT_UNLINKED {
		t.Fatal("patient should be unlinked but instead it was flagged as registered in our system")
	}

}

func createDoctorWithClinicianId(testData TestData, t *testing.T) *common.Doctor {
	signedupDoctorResponse, _, _ := SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
	_, err := testData.DB.Exec(`update doctor set clinician_id = ? where id = ?`, clinicianId, signedupDoctorResponse.DoctorId)
	if err != nil {
		t.Fatal("Unable to assign a clinicianId to the doctor: " + err.Error())
	}

	doctor, err := testData.DataApi.GetDoctorFromId(signedupDoctorResponse.DoctorId)
	if err != nil {
		t.Fatal("Unable to get doctor based on id: " + err.Error())
	}

	return doctor
}
