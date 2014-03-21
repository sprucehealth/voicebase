package integration

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_worker"
	"carefront/common"
	"carefront/libs/aws/sqs"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

	approvedRefillRequestPrescriptionId := int64(101010)
	approvedRefillAmount := int64(10)

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
			erx.LexiDrugSynId:     "1234",
			erx.LexiGenProductId:  "12345",
			erx.LexiSynonymTypeId: "123556",
			erx.NDC:               "2415",
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

	// insert erxStatusEvent for this treatment to indicate that it was sent
	_, err = testData.DB.Exec(`insert into erx_status_events (treatment_id, erx_status, creation_date, status) values (?,?,?,?)`, treatment1.Id.Int64(), api.ERX_STATUS_SENT, testTime, "ACTIVE")
	if err != nil {
		t.Fatal("Unable to insert erx_status_events x`")
	}

	// update the treatment with prescription id and pharmacy id for where prescription was routed
	_, err = testData.DB.Exec(`update treatment set erx_id = ?, pharmacy_id=? where id = ?`, treatment1.PrescriptionId.Int64(), pharmacyToReturn.LocalId, treatment1.Id.Int64())
	if err != nil {
		t.Fatal("Unable to update treatment with erx id: " + err.Error())
	}
	prescriptionIdForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
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
			PrescriptionId: common.NewObjectId(prescriptionIdForRequestedPrescription),
			ErxPharmacyId:  1234,
			DrugDBIds: map[string]string{
				erx.LexiDrugSynId:     "1234",
				erx.LexiGenProductId:  "12345",
				erx.LexiSynonymTypeId: "123556",
				erx.NDC:               "2415",
			},
			DosageStrength:       "10 mg",
			DispenseValue:        5,
			OTC:                  false,
			SubstitutionsAllowed: true,
			ErxSentDate:          &fiveMinutesBeforeTestTime,
			DoseSpotClinicianId:  clinicianId,
		},
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
			DoseSpotClinicianId:     clinicianId,
		},
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
		RefillRequestPrescriptionId:  approvedRefillRequestPrescriptionId,
		PrescriptionIdToPrescriptionStatuses: map[int64][]common.StatusEvent{
			prescriptionIdForRequestedPrescription: []common.StatusEvent{common.StatusEvent{
				Status: api.ERX_STATUS_SENT,
			},
			},
		},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

	var count int64
	err = testData.DB.QueryRow(`select count(*) from requested_treatment`).Scan(&count)
	if err != nil {
		t.Fatal("Unable to get a count for the unumber of treatments in the requested_treatment table " + err.Error())
	}
	if count == 0 {
		t.Fatalf("Expected there to be a requested treatment, but got none")
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

	if refillRequest.RequestedPrescription.OriginatingTreatmentId == 0 {
		t.Fatal("Requested prescription should be one that was found in our system, but instead its indicated to be unlinked")
	}

	if refillRequest.TreatmentPlanId == 0 {
		t.Fatal("Expected treatment plan id to be set given that the treatment is linked")
	}

	if refillRequest.Patient == nil {
		t.Fatal("Refill request expected to have patient demographics attached to it instead it doesnt")
	}

	if refillRequest.Patient.Status != api.PATIENT_REGISTERED {
		t.Fatal("Patient requesting refill expected to be in our system instead the indication is that it was an unlinked patient")
	}

	// now, lets go ahead and attempt to approve this refill request
	comment := "this is a test"
	params := url.Values{}
	params.Set("action", "approve")
	params.Set("refill_request_id", strconv.FormatInt(refillRequest.Id, 10))
	params.Set("approved_refill_amount", strconv.FormatInt(approvedRefillAmount, 10))
	params.Set("comments", comment)

	erxStatusQueue := &common.SQSQueue{}
	erxStatusQueue.QueueService = &sqs.StubSQS{}
	erxStatusQueue.QueueUrl = "local-erx"

	doctorRefillRequestsHandler := &apiservice.DoctorRefillRequestHandler{
		DataApi:        testData.DataApi,
		ErxApi:         stubErxAPI,
		ErxStatusQueue: erxStatusQueue,
	}

	ts := httptest.NewServer(doctorRefillRequestsHandler)
	defer ts.Close()

	resp, err := authPut(ts.URL, "application/x-www-form-urlencoded", bytes.NewBufferString(params.Encode()), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}

	refillRequest, err = testData.DataApi.GetRefillRequestFromId(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill request after approving request: " + err.Error())
	}

	if refillRequest.Status != api.RX_REFILL_STATUS_APPROVED {
		t.Fatalf("Expected the refill request status to be %s but was %s instead", api.RX_REFILL_STATUS_APPROVED, refillRequest.Status)
	}

	if refillRequest.ApprovedRefillAmount != approvedRefillAmount {
		t.Fatalf("Expected the approved refill amount to be %d but wsa %d instead", approvedRefillAmount, refillRequest.ApprovedRefillAmount)
	}

	if refillRequest.Comments != comment {
		t.Fatalf("Expected the comment on the refill request to be '%s' but was '%s' instead", comment, refillRequest.Comments)
	}

	// doctor queue should be empty and the denied request should be in the completed tab
	completedItems, err := testData.DataApi.GetCompletedItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get the completed items for the doctor: " + err.Error())
	}

	if len(completedItems) != 1 {
		t.Fatal("Expected there to be 1 completed item in the doctor's queue for the refill request that was just rejected")
	}

	if completedItems[0].EventType != api.EVENT_TYPE_REFILL_REQUEST || completedItems[0].ItemId != refillRequest.Id ||
		completedItems[0].Status != api.QUEUE_ITEM_STATUS_REFILL_APPROVED {
		t.Fatal("Completed item in the doctor's queue not in the expected state")
	}

	pendingItems, err = testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get the pending items for the doctor: " + err.Error())
		return
	}

	if len(pendingItems) != 0 {
		t.Fatalf("Expected there to be no pending items in the doctor's queue instead there were %d", len(pendingItems))
	}

	// attempt to consume the message put into the queue
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	// now, the status of the refill request should be Sent
	refillStatusEvents, err := testData.DataApi.GetRefillStatusEventsForRefillRequest(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill status events for refill request: " + err.Error())
	}

	if len(refillStatusEvents) != 3 {
		t.Fatalf("Expected 2 refill status events for refill request but got %d", len(refillStatusEvents))
	}

	if refillStatusEvents[0].Status != api.ERX_STATUS_SENT {
		t.Fatal("Expected the top level item for the refill request to indicate that it was successfully sent to the pharmacy")
	}
}

func TestRefillRequestComingFromDifferentPharmacyThanDispensedPrescription(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianId(testData, t)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	erxPatientId := int64(60)

	approvedRefillRequestPrescriptionId := int64(101010)
	approvedRefillAmount := int64(10)

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

	anotherPharmacyToAdd := &pharmacy.PharmacyData{
		SourceId:     "12345678",
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

	err = testData.DataApi.AddPharmacy(anotherPharmacyToAdd)
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
			erx.LexiDrugSynId:     "1234",
			erx.LexiGenProductId:  "12345",
			erx.LexiSynonymTypeId: "123556",
			erx.NDC:               "2415",
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

	// insert erxStatusEvent for this treatment to indicate that it was sent
	_, err = testData.DB.Exec(`insert into erx_status_events (treatment_id, erx_status, creation_date, status) values (?,?,?,?)`, treatment1.Id.Int64(), api.ERX_STATUS_SENT, testTime, "ACTIVE")
	if err != nil {
		t.Fatal("Unable to insert erx_status_events x`")
	}

	// update the treatment with prescription id and pharmacy id for where prescription was routed
	_, err = testData.DB.Exec(`update treatment set erx_id = ?, pharmacy_id=? where id = ?`, treatment1.PrescriptionId.Int64(), pharmacyToReturn.LocalId, treatment1.Id.Int64())
	if err != nil {
		t.Fatal("Unable to update treatment with erx id: " + err.Error())
	}

	prescriptionIdForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
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
			PrescriptionId: common.NewObjectId(prescriptionIdForRequestedPrescription),
			ErxPharmacyId:  1234,
			DrugDBIds: map[string]string{
				erx.LexiDrugSynId:     "1234",
				erx.LexiGenProductId:  "12345",
				erx.LexiSynonymTypeId: "123556",
				erx.NDC:               "2415",
			},
			DosageStrength:       "10 mg",
			DispenseValue:        5,
			OTC:                  false,
			SubstitutionsAllowed: true,
			ErxSentDate:          &fiveMinutesBeforeTestTime,
			DoseSpotClinicianId:  clinicianId,
		},
		DispensedPrescription: &common.Treatment{
			PrescriptionId:     common.NewObjectId(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      12345678,
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
			DoseSpotClinicianId:     clinicianId,
		},
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
		RefillRequestPrescriptionId:  approvedRefillRequestPrescriptionId,
		PrescriptionIdToPrescriptionStatuses: map[int64][]common.StatusEvent{
			prescriptionIdForRequestedPrescription: []common.StatusEvent{common.StatusEvent{
				Status:        api.ERX_STATUS_ERROR,
				StatusDetails: "testing this error",
			},
			},
		},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

	var count int64
	err = testData.DB.QueryRow(`select count(*) from requested_treatment`).Scan(&count)
	if err != nil {
		t.Fatal("Unable to get a count for the unumber of treatments in the requested_treatment table " + err.Error())
	}
	if count == 0 {
		t.Fatalf("Expected there to be a requested treatment, but got none")
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

	if refillRequest.RequestedPrescription.OriginatingTreatmentId == 0 {
		t.Fatal("Requested prescription should be one that was found in our system, but instead its indicated to be unlinked")
	}

	if refillRequest.Patient == nil {
		t.Fatal("Refill request expected to have patient demographics attached to it instead it doesnt")
	}

	if refillRequest.Patient.Status != api.PATIENT_REGISTERED {
		t.Fatal("Patient requesting refill expected to be in our system instead the indication is that it was an unlinked patient")
	}

	if refillRequest.RequestedPrescription.Pharmacy == nil || refillRequest.DispensedPrescription.Pharmacy == nil {
		t.Fatal("Expected pharmacy object to be present for requested and dispensed prescriptions")
	}

	if refillRequest.RequestedPrescription.Pharmacy.SourceId == refillRequest.DispensedPrescription.Pharmacy.SourceId {
		t.Fatal("Expected the pharmacies to be different between the requested and the dispensed prescriptions")
	}

	erxStatusQueue := &common.SQSQueue{}
	erxStatusQueue.QueueService = &sqs.StubSQS{}
	erxStatusQueue.QueueUrl = "local-erx"

	// now, lets go ahead and attempt to approve this refill request
	comment := "this is a test"
	params := url.Values{}
	params.Set("action", "approve")
	params.Set("refill_request_id", strconv.FormatInt(refillRequest.Id, 10))
	params.Set("approved_refill_amount", strconv.FormatInt(approvedRefillAmount, 10))
	params.Set("comments", comment)

	doctorRefillRequestsHandler := &apiservice.DoctorRefillRequestHandler{
		DataApi:        testData.DataApi,
		ErxApi:         stubErxAPI,
		ErxStatusQueue: erxStatusQueue,
	}

	ts := httptest.NewServer(doctorRefillRequestsHandler)
	defer ts.Close()

	resp, err := authPut(ts.URL, "application/x-www-form-urlencoded", bytes.NewBufferString(params.Encode()), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}

	refillRequest, err = testData.DataApi.GetRefillRequestFromId(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill request after approving request: " + err.Error())
	}

	if refillRequest.Status != api.RX_REFILL_STATUS_APPROVED {
		t.Fatalf("Expected the refill request status to be %s but was %s instead", api.RX_REFILL_STATUS_APPROVED, refillRequest.Status)
	}

	if refillRequest.ApprovedRefillAmount != approvedRefillAmount {
		t.Fatalf("Expected the approved refill amount to be %d but wsa %d instead", approvedRefillAmount, refillRequest.ApprovedRefillAmount)
	}

	if refillRequest.Comments != comment {
		t.Fatalf("Expected the comment on the refill request to be '%s' but was '%s' instead", comment, refillRequest.Comments)
	}

	// doctor queue should be empty and the denied request should be in the completed tab
	completedItems, err := testData.DataApi.GetCompletedItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get the completed items for the doctor: " + err.Error())
	}

	if len(completedItems) != 1 {
		t.Fatal("Expected there to be 1 completed item in the doctor's queue for the refill request that was just rejected")
	}

	if completedItems[0].EventType != api.EVENT_TYPE_REFILL_REQUEST || completedItems[0].ItemId != refillRequest.Id ||
		completedItems[0].Status != api.QUEUE_ITEM_STATUS_REFILL_APPROVED {
		t.Fatal("Completed item in the doctor's queue not in the expected state")
	}

	pendingItems, err = testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get the pending items for the doctor: " + err.Error())
		return
	}

	if len(pendingItems) != 0 {
		t.Fatalf("Expected there to be no pending items in the doctor's queue instead there were %d", len(pendingItems))
	}

	// attempt to consume the message put into the queue
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	// now, the status of the refill request should be Sent
	refillStatusEvents, err := testData.DataApi.GetRefillStatusEventsForRefillRequest(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill status events for refill request: " + err.Error())
	}

	if len(refillStatusEvents) != 3 {
		t.Fatalf("Expected 2 refill status events for refill request but got %d", len(refillStatusEvents))
	}

	if refillStatusEvents[0].Status != api.ERX_STATUS_ERROR {
		t.Fatal("Expected the top level item for the refill request to indicate that it was successfully sent to the pharmacy")
	}

	if refillStatusEvents[0].StatusDetails == "" {
		t.Fatal("Expected there be to an error message for the refill request  given that there was an errror sending to pharmacy")
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
	prescriptionIdForRequestedPrescription := int64(5504)
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
			PrescriptionId:     common.NewObjectId(prescriptionIdForRequestedPrescription),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      123,
			DrugDBIds: map[string]string{
				erx.LexiDrugSynId:     "1234",
				erx.LexiGenProductId:  "12345",
				erx.LexiSynonymTypeId: "123556",
				erx.NDC:               "2415",
			},
			DrugName:                "Teting (This - Drug)",
			DosageStrength:          "10 mg",
			DispenseValue:           5,
			DispenseUnitDescription: "Tablet",
			NumberRefills:           5,
			SubstitutionsAllowed:    false,
			DaysSupply:              10,
			PatientInstructions:     "Take once daily",
			ErxSentDate:             &testTime,
			OTC:                     false,
			DoseSpotClinicianId:     clinicianId,
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
			ErxSentDate:             &testTime,
			OTC:                     false,
			DoseSpotClinicianId:     clinicianId,
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
		PrescriptionIdToPrescriptionStatuses: map[int64][]common.StatusEvent{
			prescriptionIdForRequestedPrescription: []common.StatusEvent{common.StatusEvent{
				Status: api.ERX_STATUS_DELETED,
			},
			},
		},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

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

	if refillRequest.RequestedPrescription.OriginatingTreatmentId != 0 {
		t.Fatal("Requested prescription should be unlinked but was instead found in the system")
	}

	if refillRequest.Patient == nil {
		t.Fatal("Refill request expected to have patient demographics attached to it instead it doesnt")
	}

	if refillRequest.Patient.Status != api.PATIENT_REGISTERED {
		t.Fatal("Patient requesting refill expected to be in our system instead the indication is that it was an unlinked patient")
	}

	denialReasons, err := testData.DataApi.GetRefillRequestDenialReasons()
	if err != nil || len(denialReasons) == 0 {
		t.Fatal("Unable to get the denial reasons for the refill request")
	}

	erxStatusQueue := &common.SQSQueue{}
	erxStatusQueue.QueueService = &sqs.StubSQS{}
	erxStatusQueue.QueueUrl = "local-erx"

	// now, lets go ahead and attempt to deny this refill request
	comment := "this is a test"
	params := url.Values{}
	params.Set("action", "deny")
	params.Set("denial_reason_id", strconv.FormatInt(denialReasons[0].Id, 10))
	params.Set("comments", comment)
	params.Set("refill_request_id", strconv.FormatInt(refillRequest.Id, 10))

	doctorRefillRequestsHandler := &apiservice.DoctorRefillRequestHandler{
		DataApi:        testData.DataApi,
		ErxApi:         stubErxAPI,
		ErxStatusQueue: erxStatusQueue,
	}

	ts := httptest.NewServer(doctorRefillRequestsHandler)
	defer ts.Close()

	resp, err := authPut(ts.URL, "application/x-www-form-urlencoded", bytes.NewBufferString(params.Encode()), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}

	refillRequest, err = testData.DataApi.GetRefillRequestFromId(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill request after approving request: " + err.Error())
	}

	if refillRequest.Status != api.RX_REFILL_STATUS_DENIED {
		t.Fatalf("Expected the refill request status to be %s but was %s instead", api.RX_REFILL_STATUS_DENIED, refillRequest.Status)
	}

	if refillRequest.Comments != comment {
		t.Fatalf("Expected the comment on the refill request to be '%s' but was '%s' instead", comment, refillRequest.Comments)
	}

	if refillRequest.DenialReason != denialReasons[0].DenialReason {
		t.Fatalf("Denial reason expected to be '%s' but is '%s' instead", denialReasons[0].DenialReason, refillRequest.DenialReason)
	}

	// doctor queue should be empty and the denied request should be in the completed tab
	completedItems, err := testData.DataApi.GetCompletedItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get the completed items for the doctor: " + err.Error())
	}

	if len(completedItems) != 1 {
		t.Fatal("Expected there to be 1 completed item in the doctor's queue for the refill request that was just rejected")
	}

	if completedItems[0].EventType != api.EVENT_TYPE_REFILL_REQUEST || completedItems[0].ItemId != refillRequest.Id ||
		completedItems[0].Status != api.QUEUE_ITEM_STATUS_REFILL_DENIED {
		t.Fatal("Completed item in the doctor's queue not in the expected state")
	}

	pendingItems, err = testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get the pending items for the doctor: " + err.Error())
		return
	}

	if len(pendingItems) != 0 {
		t.Fatalf("Expected there to be no pending items in the doctor's queue instead there were %d", len(pendingItems))
	}

	// attempt to consume the message put into the queue
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	// now, the status of the refill request should be Sent
	refillStatusEvents, err := testData.DataApi.GetRefillStatusEventsForRefillRequest(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill status events for refill request: " + err.Error())
	}

	if len(refillStatusEvents) != 3 {
		t.Fatalf("Expected 2 refill status events for refill request but got %d", len(refillStatusEvents))
	}

	if refillStatusEvents[0].Status != api.ERX_STATUS_DELETED {
		t.Fatal("Expected the top level item for the refill request to indicate that it was successfully sent to the pharmacy")
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
				erx.LexiDrugSynId:     "1234",
				erx.LexiGenProductId:  "12345",
				erx.LexiSynonymTypeId: "123556",
				erx.NDC:               "2415",
			},
			DrugName:                "Teting (This - Drug)",
			DosageStrength:          "10 mg",
			DispenseValue:           5,
			DispenseUnitDescription: "Tablet",
			NumberRefills:           5,
			SubstitutionsAllowed:    false,
			DaysSupply:              10,
			PatientInstructions:     "Take once daily",
			ErxSentDate:             &testTime,
			OTC:                     false,
			DoseSpotClinicianId:     clinicianId,
		},
		DispensedPrescription: &common.Treatment{
			PrescriptionId:     common.NewObjectId(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      123,
			DrugDBIds: map[string]string{
				erx.LexiDrugSynId:     "1234",
				erx.LexiGenProductId:  "12345",
				erx.LexiSynonymTypeId: "123556",
				erx.NDC:               "2415",
			},
			DrugName:                "Teting (This - Drug)",
			DosageStrength:          "10 mg",
			DispenseValue:           5,
			DispenseUnitDescription: "Tablet",
			NumberRefills:           5,
			SubstitutionsAllowed:    false,
			DaysSupply:              10,
			PatientInstructions:     "Take once daily",
			ErxSentDate:             &testTime,
			OTC:                     false,
			DoseSpotClinicianId:     clinicianId,
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
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

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

	if refillRequest.RequestedPrescription.OriginatingTreatmentId != 0 {
		t.Fatal("Requested prescription should be unlinked but was instead found in the system")
	}

	if refillRequest.Patient == nil {
		t.Fatal("Refill request expected to have patient demographics attached to it instead it doesnt")
	}

	if refillRequest.Patient.Status != api.PATIENT_UNLINKED {
		t.Fatal("patient should be unlinked but instead it was flagged as registered in our system")
	}

	if refillRequest.RequestedPrescription.Doctor == nil || refillRequest.DispensedPrescription.Doctor == nil {
		t.Fatal("Expected doctor object to be present for the requested and dispensed prescription")
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
