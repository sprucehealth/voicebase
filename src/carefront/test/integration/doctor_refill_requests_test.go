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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
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
		OTC:                     false,
		ERx: &common.ERxData{
			PrescriptionId:     common.NewObjectId(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      1234,
			PharmacyLocalId:    common.NewObjectId(pharmacyToReturn.LocalId),
			ErxSentDate:        &testTime,
		},
	}

	// add this treatment to the treatment plan
	err = testData.DataApi.AddTreatmentsForPatientVisit([]*common.Treatment{treatment1}, doctor.DoctorId.Int64(), treatmentPlanId, signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to add treatment for patient visit: " + err.Error())
	}

	// insert erxStatusEvent for this treatment to indicate that it was sent
	_, err = testData.DB.Exec(`insert into erx_status_events (treatment_id, erx_status, creation_date, status) values (?,?,?,?)`, treatment1.Id.Int64(), api.ERX_STATUS_SENT, testTime, "ACTIVE")
	if err != nil {
		t.Fatal("Unable to insert erx_status_events x`")
	}

	// update the treatment with prescription id and pharmacy id for where prescription was routed
	_, err = testData.DB.Exec(`update treatment set erx_id = ?, pharmacy_id=? where id = ?`, treatment1.ERx.PrescriptionId.Int64(), pharmacyToReturn.LocalId, treatment1.Id.Int64())
	if err != nil {
		t.Fatal("Unable to update treatment with erx id: " + err.Error())
	}
	prescriptionIdForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemId := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemId:      refillRequestQueueItemId,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientId:              erxPatientId,
		PatientAddedForRequest:    false,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
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
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianId: clinicianId,
				PrescriptionId:      common.NewObjectId(prescriptionIdForRequestedPrescription),
				ErxPharmacyId:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       1234,
				DoseSpotClinicianId: clinicianId,
			},
		},
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
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

	if refillRequestStatuses[0].ItemId != refillRequestItem.Id ||
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
		pendingItems[0].ItemId != refillRequestStatuses[0].ItemId {
		t.Fatal("Pending item found in the doctor's queue is not the expected item")
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
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
}

func TestApproveRefillRequestAndSuccessfulSendToPharmacy(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianId(testData, t)

	approvedRefillRequestPrescriptionId := int64(101010)
	approvedRefillAmount := int64(10)

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

	err := testData.DataApi.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIdForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemId := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemId:      refillRequestQueueItemId,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientId:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
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
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianId: clinicianId,
				PrescriptionId:      common.NewObjectId(prescriptionIdForRequestedPrescription),
				ErxPharmacyId:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				DoseSpotClinicianId: clinicianId,
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       1234,
			},
		},
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		PatientDetailsToReturn:       patientToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
		RefillRequestPrescriptionIds: map[int64]int64{
			refillRequestQueueItemId: approvedRefillRequestPrescriptionId,
		},
		PrescriptionIdToPrescriptionStatuses: map[int64][]common.StatusEvent{
			approvedRefillRequestPrescriptionId: []common.StatusEvent{common.StatusEvent{
				Status: api.ERX_STATUS_SENT,
			},
			},
		},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

	refillRequestStatuses, err := testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	// lets go ahead and attempt to approve this refill request
	comment := "this is a test"

	requestData := apiservice.DoctorRefillRequestRequestData{
		RefillRequestId:      common.NewObjectId(refillRequest.Id),
		Action:               "approve",
		ApprovedRefillAmount: 10,
		Comments:             comment,
	}

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

	// sleep for a brief moment before approving so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatal("Unable to marshal json object: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successful request to approve refill request: ")
	}

	refillRequest, err = testData.DataApi.GetRefillRequestFromId(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill request after approving request: " + err.Error())
	}

	if len(refillRequest.RxHistory) != 2 {
		t.Fatalf("Expected 2 items in the rx history for the refill request instead got %d", len(refillRequest.RxHistory))
	}

	if refillRequest.RxHistory[0].Status != api.RX_REFILL_STATUS_APPROVED {
		t.Fatalf("Expected the refill request status to be %s but was %s instead: %+v", api.RX_REFILL_STATUS_APPROVED, refillRequest.RxHistory[0].Status, refillRequest.RxHistory)
	}

	if refillRequest.ApprovedRefillAmount != approvedRefillAmount {
		t.Fatalf("Expected the approved refill amount to be %d but wsa %d instead", approvedRefillRequestPrescriptionId, refillRequest.ApprovedRefillAmount)
	}

	if refillRequest.Comments != comment {
		t.Fatalf("Expected the comment on the refill request to be '%s' but was '%s' instead", comment, refillRequest.Comments)
	}

	if refillRequest.PrescriptionId != approvedRefillRequestPrescriptionId {
		t.Fatalf("Expected the prescription id returned to be %d but instead it was %d", approvedRefillAmount, refillRequest.PrescriptionId)
	}

	// doctor queue should be empty and the approved request should be in the completed tab
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

	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get the pending items for the doctor: " + err.Error())
		return
	}

	if len(pendingItems) != 0 {
		t.Fatalf("Expected there to be no pending items in the doctor's queue instead there were %d", len(pendingItems))
	}

	// sleep for a brief moment before approving so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	// attempt to consume the message put into the queue
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	// now, the status of the refill request should be Sent
	refillStatusEvents, err := testData.DataApi.GetRefillStatusEventsForRefillRequest(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill status events for refill request: " + err.Error())
	}

	if len(refillStatusEvents) != 3 {
		t.Fatalf("Expected 3 refill status events for refill request but got %d", len(refillStatusEvents))
	}

	if refillStatusEvents[0].Status != api.RX_REFILL_STATUS_SENT {
		t.Fatal("Expected the top level item for the refill request to indicate that it was successfully sent to the pharmacy")
	}
}
func TestApproveRefillRequestAndErrorSendingToPharmacy(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianId(testData, t)

	approvedRefillRequestPrescriptionId := int64(101010)
	approvedRefillAmount := int64(10)

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

	err := testData.DataApi.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIdForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemId := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemId:      refillRequestQueueItemId,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientId:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
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
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianId: clinicianId,
				PrescriptionId:      common.NewObjectId(prescriptionIdForRequestedPrescription),
				ErxPharmacyId:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
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

			OTC: false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				DoseSpotClinicianId: clinicianId,
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       1234,
			},
		},
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		PatientDetailsToReturn:       patientToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
		RefillRequestPrescriptionIds: map[int64]int64{
			refillRequestQueueItemId: approvedRefillRequestPrescriptionId,
		}, PrescriptionIdToPrescriptionStatuses: map[int64][]common.StatusEvent{
			approvedRefillRequestPrescriptionId: []common.StatusEvent{common.StatusEvent{
				Status:        api.ERX_STATUS_ERROR,
				StatusDetails: "testing this error",
			},
			},
		},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

	refillRequestStatuses, err := testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	// lets go ahead and attempt to approve this refill request
	comment := "this is a test"

	requestData := apiservice.DoctorRefillRequestRequestData{
		RefillRequestId:      common.NewObjectId(refillRequest.Id),
		Action:               "approve",
		ApprovedRefillAmount: 10,
		Comments:             comment,
	}

	erxStatusQueue := &common.SQSQueue{}
	erxStatusQueue.QueueService = &sqs.StubSQS{}
	erxStatusQueue.QueueUrl = "local-erx"

	doctorRefillRequestsHandler := &apiservice.DoctorRefillRequestHandler{
		DataApi:        testData.DataApi,
		ErxApi:         stubErxAPI,
		ErxStatusQueue: erxStatusQueue,
	}

	// sleep for a brief moment before approving so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	ts := httptest.NewServer(doctorRefillRequestsHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatal("Unable to marshal json object: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successful request to approve refill request: ")
	}

	refillRequest, err = testData.DataApi.GetRefillRequestFromId(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill request after approving request: " + err.Error())
	}

	if len(refillRequest.RxHistory) != 2 {
		t.Fatalf("Expected 2 items in the rx history for the refill request instead got %d", len(refillRequest.RxHistory))
	}

	if refillRequest.RxHistory[0].Status != api.RX_REFILL_STATUS_APPROVED {
		t.Fatalf("Expected the refill request status to be %s but was %s instead: %+v", api.RX_REFILL_STATUS_APPROVED, refillRequest.RxHistory[0].Status, refillRequest.RxHistory)
	}

	if refillRequest.ApprovedRefillAmount != approvedRefillAmount {
		t.Fatalf("Expected the approved refill amount to be %d but wsa %d instead", approvedRefillAmount, refillRequest.ApprovedRefillAmount)
	}

	if refillRequest.Comments != comment {
		t.Fatalf("Expected the comment on the refill request to be '%s' but was '%s' instead", comment, refillRequest.Comments)
	}

	if refillRequest.PrescriptionId != approvedRefillRequestPrescriptionId {
		t.Fatalf("Expected the prescription id returned to be %d but instead it was %d", approvedRefillRequestPrescriptionId, refillRequest.PrescriptionId)
	}

	// doctor queue should be empty and the approved request should be in the completed tab
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

	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get the pending items for the doctor: " + err.Error())
		return
	}

	if len(pendingItems) != 0 {
		t.Fatalf("Expected there to be no pending items in the doctor's queue instead there were %d", len(pendingItems))
	}

	// sleep for a brief moment before approving so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	// attempt to consume the message put into the queue
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	refillStatusEvents, err := testData.DataApi.GetRefillStatusEventsForRefillRequest(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill status events for refill request: " + err.Error())
	}

	if len(refillStatusEvents) != 3 {
		t.Fatalf("Expected 3 refill status events for refill request but got %d", len(refillStatusEvents))
	}

	if refillStatusEvents[0].Status != api.RX_REFILL_STATUS_ERROR {
		t.Fatal("Expected the top level item for the refill request to indicate that it was successfully sent to the pharmacy")
	}

	if refillStatusEvents[0].StatusDetails == "" {
		t.Fatal("Expected there be to an error message for the refill request  given that there was an errror sending to pharmacy")
	}

	// lets make sure that the error for the refill request makes it into the doctor's queue
	pendingItems, err = testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get pending items in doctors queue: " + err.Error())
	}

	if len(pendingItems) != 1 {
		t.Fatalf("Expected there to be 1 item in the doctors queue but there were %d", len(pendingItems))
	}

	if pendingItems[0].EventType != api.EVENT_TYPE_REFILL_TRANSMISSION_ERROR {
		t.Fatalf("Expected the 1 item in teh doctors queue to be a transmission error for a refill request but instead it was %s", pendingItems[0].EventType)
	}

	// lets go ahead and resolve this error
	doctorPrescriptionErrorIgnoreHandler := &apiservice.DoctorPrescriptionErrorIgnoreHandler{
		DataApi: testData.DataApi,
		ErxApi:  stubErxAPI,
	}

	// sleep for a brief moment before approving so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	params := url.Values{}
	params.Set("refill_request_id", fmt.Sprintf("%d", refillRequest.Id))

	errorIgnoreTs := httptest.NewServer(doctorPrescriptionErrorIgnoreHandler)
	resp, err = authPost(errorIgnoreTs.URL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()), doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to resolve refill request transmission error: %+v", err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to successfull resolve refill request transmission error", t)

	// check the rx history of the refill request
	refillRequest, err = testData.DataApi.GetRefillRequestFromId(refillRequest.Id)
	if err != nil {
		t.Fatalf("Unable to get refill request : %+v", refillRequest)
	}

	if len(refillRequest.RxHistory) != 4 {
		t.Fatalf("Expected refill request to have 4 events in its history, instead it had %d", len(refillRequest.RxHistory))
	}

	if refillRequest.RxHistory[0].Status != api.RX_REFILL_STATUS_ERROR_RESOLVED {
		t.Fatal("Expected the refill request to be resolved once the doctor resolved the error")
	}

	pendingItems, err = testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatalf("there should be no pending items in the doctor queue: %+v", err)
	}

	if len(pendingItems) != 0 {
		t.Fatalf("Expected to have no items in the doctor queue, instead have %d", len(pendingItems))
	}
}

func TestDenyRefillRequestAndSuccessfulDelete(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianId(testData, t)

	deniedRefillRequestPrescriptionId := int64(101010)

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

	err := testData.DataApi.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIdForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemId := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemId:      refillRequestQueueItemId,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientId:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
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
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianId: clinicianId,
				PrescriptionId:      common.NewObjectId(prescriptionIdForRequestedPrescription),
				ErxPharmacyId:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       1234,
				DoseSpotClinicianId: clinicianId,
			},
		},
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		PatientDetailsToReturn:       patientToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
		RefillRequestPrescriptionIds: map[int64]int64{
			refillRequestQueueItemId: deniedRefillRequestPrescriptionId,
		}, PrescriptionIdToPrescriptionStatuses: map[int64][]common.StatusEvent{
			deniedRefillRequestPrescriptionId: []common.StatusEvent{common.StatusEvent{
				Status: api.ERX_STATUS_DELETED,
			},
			},
		},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

	refillRequestStatuses, err := testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
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
	requestData := apiservice.DoctorRefillRequestRequestData{
		RefillRequestId: common.NewObjectId(refillRequest.Id),
		Action:          "deny",
		DenialReasonId:  common.NewObjectId(denialReasons[0].Id),
		Comments:        comment,
	}

	doctorRefillRequestsHandler := &apiservice.DoctorRefillRequestHandler{
		DataApi:        testData.DataApi,
		ErxApi:         stubErxAPI,
		ErxStatusQueue: erxStatusQueue,
	}

	// sleep for a brief moment before denyingh so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	ts := httptest.NewServer(doctorRefillRequestsHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		t.Fatal("Unable to marshal json into object: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
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

	if len(refillRequest.RxHistory) != 2 {
		t.Fatalf("Expected two items in the rx history of the refill request instead got %d", len(refillRequest.RxHistory))
	}

	if refillRequest.RxHistory[0].Status != api.RX_REFILL_STATUS_DENIED {
		t.Fatalf("Expected the refill request status to be %s but was %s instead: %+v", api.RX_REFILL_STATUS_DENIED, refillRequest.RxHistory[0].Status, refillRequest.RxHistory)
	}

	if refillRequest.Comments != comment {
		t.Fatalf("Expected the comment on the refill request to be '%s' but was '%s' instead", comment, refillRequest.Comments)
	}

	if refillRequest.PrescriptionId != deniedRefillRequestPrescriptionId {
		t.Fatalf("Expected the prescription id returned to be %d but instead it was %d", deniedRefillRequestPrescriptionId, refillRequest.PrescriptionId)
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

	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal("Unable to get the pending items for the doctor: " + err.Error())
		return
	}

	if len(pendingItems) != 0 {
		t.Fatalf("Expected there to be no pending items in the doctor's queue instead there were %d", len(pendingItems))
	}

	// sleep for a brief moment before approving so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	// attempt to consume the message put into the queue
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	// now, the status of the refill request should be Sent
	refillStatusEvents, err := testData.DataApi.GetRefillStatusEventsForRefillRequest(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill status events for refill request: " + err.Error())
	}

	if len(refillStatusEvents) != 3 {
		t.Fatalf("Expected 3 refill status events for refill request but got %d", len(refillStatusEvents))
	}

	if refillStatusEvents[0].Status != api.RX_REFILL_STATUS_DELETED {
		t.Fatal("Expected the top level item for the refill request to indicate that it was successfully sent to the pharmacy")
	}
}

func TestDenyRefillRequestWithDNTFWithoutTreatment(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianId(testData, t)

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

	err := testData.DataApi.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIdForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemId := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemId:      refillRequestQueueItemId,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientId:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
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
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianId: clinicianId,
				PrescriptionId:      common.NewObjectId(prescriptionIdForRequestedPrescription),
				ErxPharmacyId:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       1234,
				DoseSpotClinicianId: clinicianId,
			},
		},
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		PatientDetailsToReturn:       patientToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

	refillRequestStatuses, err := testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	denialReasons, err := testData.DataApi.GetRefillRequestDenialReasons()
	if err != nil || len(denialReasons) == 0 {
		t.Fatal("Unable to get the denial reasons for the refill request")
	}

	var dntfReason *api.RefillRequestDenialReason
	for _, denialReason := range denialReasons {
		if denialReason.DenialCode == api.RX_REFILL_DNTF_REASON_CODE {
			dntfReason = denialReason
			break
		}
	}

	if dntfReason == nil {
		t.Fatal("Unable to find DNTF reason in database: " + err.Error())
	}

	erxStatusQueue := &common.SQSQueue{}
	erxStatusQueue.QueueService = &sqs.StubSQS{}
	erxStatusQueue.QueueUrl = "local-erx"

	// now, lets go ahead and attempt to deny this refill request
	comment := "this is a test"
	requestData := apiservice.DoctorRefillRequestRequestData{
		RefillRequestId: common.NewObjectId(refillRequest.Id),
		Action:          "deny",
		DenialReasonId:  common.NewObjectId(dntfReason.Id),
		Comments:        comment,
	}

	doctorRefillRequestsHandler := &apiservice.DoctorRefillRequestHandler{
		DataApi:        testData.DataApi,
		ErxApi:         stubErxAPI,
		ErxStatusQueue: erxStatusQueue,
	}

	// sleep for a brief moment before denyingh so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	ts := httptest.NewServer(doctorRefillRequestsHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		t.Fatal("Unable to marshal json into object: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected %d due to missing treatment object instead got %d ", http.StatusBadRequest, resp.StatusCode)
	}

	errorResponse := apiservice.ErrorResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		t.Fatal("Unable to unmarshal response body into json object: " + err.Error())
	}

	if errorResponse.DeveloperCode != apiservice.DEVELOPER_TREATMENT_MISSING_DNTF {
		t.Fatalf("Expected developer code of %d instead got %d", apiservice.DEVELOPER_TREATMENT_MISSING_DNTF, errorResponse.DeveloperCode)
	}

}

func setUpDeniedRefillRequestWithDNTF(t *testing.T, testData TestData, endErxStatus string, toAddTemplatedTreatment bool) *common.Treatment {
	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianId(testData, t)

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

	err := testData.DataApi.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	comment := "this is a test"
	treatmentToAdd := common.Treatment{
		DrugInternalName: "Testing (If - This Works)",
		DrugDBIds: map[string]string{
			erx.LexiSynonymTypeId: "12345",
			erx.LexiDrugSynId:     "123151",
			erx.LexiGenProductId:  "124151",
			erx.NDC:               "1415",
		},
		DosageStrength:      "10 mg",
		DispenseValue:       1,
		DispenseUnitId:      common.NewObjectId(12),
		NumberRefills:       1,
		OTC:                 false,
		PatientInstructions: "patient instructions",
	}

	if toAddTemplatedTreatment {

		treatmentTemplate := &common.DoctorTreatmentTemplate{}
		treatmentTemplate.Name = "Favorite Treatment #1"
		treatmentTemplate.Treatment = &treatmentToAdd

		doctorFavoriteTreatmentsHandler := &apiservice.DoctorTreatmentTemplatesHandler{DataApi: testData.DataApi}
		ts := httptest.NewServer(doctorFavoriteTreatmentsHandler)
		defer ts.Close()

		treatmentTemplatesRequest := &apiservice.DoctorTreatmentTemplatesRequest{TreatmentTemplates: []*common.DoctorTreatmentTemplate{treatmentTemplate}}
		data, err := json.Marshal(&treatmentTemplatesRequest)
		if err != nil {
			t.Fatal("Unable to marshal request body for adding treatments to patient visit")
		}

		resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(data), doctor.AccountId.Int64())
		if err != nil {
			t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Request to add treatments failed with http status code %d", resp.StatusCode)
		}

		treatmentTemplatesResponse := &apiservice.DoctorTreatmentTemplatesResponse{}
		err = json.NewDecoder(resp.Body).Decode(treatmentTemplatesResponse)
		if err != nil {
			t.Fatal("Unable to unmarshal response into object : " + err.Error())
		}

		treatmentToAdd.DoctorTreatmentTemplateId = treatmentTemplatesResponse.TreatmentTemplates[0].Id
	}

	testTime := time.Now()

	prescriptionIdForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemId := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemId:      refillRequestQueueItemId,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientId:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
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
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianId: clinicianId,
				PrescriptionId:      common.NewObjectId(prescriptionIdForRequestedPrescription),
				ErxPharmacyId:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       1234,
				DoseSpotClinicianId: clinicianId,
			},
		},
	}

	prescriptionIdForTreatment := int64(1234151515)
	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		PatientDetailsToReturn:       patientToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
		PrescriptionIdsToReturn:      []int64{prescriptionIdForTreatment},
		SelectedMedicationToReturn:   &common.Treatment{},
		PrescriptionIdToPrescriptionStatuses: map[int64][]common.StatusEvent{
			prescriptionIdForTreatment: []common.StatusEvent{common.StatusEvent{
				Status: endErxStatus,
			},
			},
		},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

	refillRequestStatuses, err := testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	denialReasons, err := testData.DataApi.GetRefillRequestDenialReasons()
	if err != nil || len(denialReasons) == 0 {
		t.Fatal("Unable to get the denial reasons for the refill request")
	}

	var dntfReason *api.RefillRequestDenialReason
	for _, denialReason := range denialReasons {
		if denialReason.DenialCode == api.RX_REFILL_DNTF_REASON_CODE {
			dntfReason = denialReason
			break
		}
	}

	if dntfReason == nil {
		t.Fatal("Unable to find DNTF reason in database: " + err.Error())
	}

	erxStatusQueue := &common.SQSQueue{}
	erxStatusQueue.QueueService = &sqs.StubSQS{}
	erxStatusQueue.QueueUrl = "local-erx"

	// now, lets go ahead and attempt to deny this refill request

	requestData := apiservice.DoctorRefillRequestRequestData{
		RefillRequestId: common.NewObjectId(refillRequest.Id),
		Action:          "deny",
		DenialReasonId:  common.NewObjectId(dntfReason.Id),
		Comments:        comment,
		Treatment:       &treatmentToAdd,
	}

	doctorRefillRequestsHandler := &apiservice.DoctorRefillRequestHandler{
		DataApi:        testData.DataApi,
		ErxApi:         stubErxAPI,
		ErxStatusQueue: erxStatusQueue,
	}

	// sleep for a brief moment before denyingh so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	ts := httptest.NewServer(doctorRefillRequestsHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		t.Fatal("Unable to marshal json into object: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to deny refill request: " + err.Error())
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful request to deny refill request: "+string(respBody), t)

	// get refill request to ensure that it was denied
	refillRequest, err = testData.DataApi.GetRefillRequestFromId(refillRequest.Id)
	if err != nil {
		t.Fatalf("Unable to get refill request from id: %+v", err)
	}

	if len(refillRequest.RxHistory) != 2 {
		t.Fatalf("Expected there to be 2 refill request events instead there were %d", len(refillRequest.RxHistory))
	}

	if refillRequest.RxHistory[0].Status != api.RX_REFILL_STATUS_DENIED {
		t.Fatalf("Expected top level refill request status of %s instead got %s", refillRequestItem.RxHistory[0].Status, api.RX_REFILL_STATUS_DENIED)
	}

	// get unlinked treatment
	unlinkedDNTFTreatmentStatusEvents, err := testData.DataApi.GetErxStatusEventsForDNTFTreatmentBasedOnPatientId(refillRequest.Patient.PatientId.Int64())
	if err != nil {
		t.Fatalf("Unable to get status events for dntf treatment: %+v", err)
	}

	if len(unlinkedDNTFTreatmentStatusEvents) != 2 {
		t.Fatalf("Expected 2 status events for unlinked dntf treatments instead got %d", len(unlinkedDNTFTreatmentStatusEvents))
	}

	unlinkedTreatment, err := testData.DataApi.GetUnlinkedDNTFTreatment(unlinkedDNTFTreatmentStatusEvents[0].ItemId)
	if err != nil {
		t.Fatalf("Unable to get treatments pertaining to patient: %+v", err)
	}

	if unlinkedTreatment.ERx.PrescriptionId.Int64() != prescriptionIdForTreatment {
		t.Fatal("Expected the treatment to have the prescription id set as was expected")
	}

	if unlinkedTreatment.ERx.Pharmacy.LocalId != refillRequest.RequestedPrescription.ERx.Pharmacy.LocalId {
		t.Fatalf("Expected the new rx to be sent to the same pharmacy as the requestd prescription in the refill request which was not the case. New rx was sent to %d while requested prescription was sent to %d",
			unlinkedTreatment.ERx.Pharmacy.LocalId, refillRequest.RequestedPrescription.ERx.Pharmacy.LocalId)
	}

	if len(unlinkedTreatment.ERx.RxHistory) != 2 {
		t.Fatalf("Expected there to exist 1 status event pertaining to DNTF but instead got %d", len(unlinkedTreatment.ERx.RxHistory))
	}

	for _, unlinkedTreatmentStatusEvent := range unlinkedTreatment.ERx.RxHistory {
		if unlinkedTreatmentStatusEvent.InternalStatus == api.STATUS_INACTIVE && unlinkedTreatmentStatusEvent.Status != api.ERX_STATUS_NEW_RX_FROM_DNTF {
			t.Fatalf("Expected top level item in rx history to be %s instead it was %s", api.ERX_STATUS_NEW_RX_FROM_DNTF, unlinkedTreatmentStatusEvent.Status)
		}
	}

	// check dntf mapping to ensure that there is an entry
	var dntfMappingCount int64
	if err = testData.DB.QueryRow(`select count(*) from dntf_mapping`).Scan(&dntfMappingCount); err != nil {
		t.Fatalf("Unable to count number of entries in dntf mapping table: %+v", err)
	}

	if dntfMappingCount != 1 {
		t.Fatalf("Expected 1 entry in dntf mapping table instead got %d", dntfMappingCount)
	}

	// check erx status to be sent once its sent
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	unlinkedTreatment, err = testData.DataApi.GetUnlinkedDNTFTreatment(unlinkedTreatment.Id.Int64())
	if err != nil {
		t.Fatalf("Unable to get unlinked dntf treatment: %+v", err)
	}

	return unlinkedTreatment
}

func TestDenyRefillRequestWithDNTFWithUnlinkedTreatment(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	unlinkedTreatment := setUpDeniedRefillRequestWithDNTF(t, testData, api.ERX_STATUS_SENT, false)

	if len(unlinkedTreatment.ERx.RxHistory) != 3 {
		t.Fatalf("Expcted 3 events from rx history of unlinked treatment instead got %d", len(unlinkedTreatment.ERx.RxHistory))
	}

	for _, unlinkedTreatmentStatusEvent := range unlinkedTreatment.ERx.RxHistory {
		if unlinkedTreatmentStatusEvent.InternalStatus == api.STATUS_ACTIVE && unlinkedTreatmentStatusEvent.Status != api.ERX_STATUS_SENT {
			t.Fatalf("Expected status %s for top level status of unlinked treatment but got %s", api.ERX_STATUS_SENT, unlinkedTreatmentStatusEvent.Status)
		}
	}
}

func TestDenyRefillRequestWithDNTFWithUnlinkedTreatmentFromTemplatedTreatment(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	unlinkedTreatment := setUpDeniedRefillRequestWithDNTF(t, testData, api.ERX_STATUS_SENT, true)

	if len(unlinkedTreatment.ERx.RxHistory) != 3 {
		t.Fatalf("Expcted 3 events from rx history of unlinked treatment instead got %d", len(unlinkedTreatment.ERx.RxHistory))
	}

	for _, unlinkedTreatmentStatusEvent := range unlinkedTreatment.ERx.RxHistory {
		if unlinkedTreatmentStatusEvent.InternalStatus == api.STATUS_ACTIVE && unlinkedTreatmentStatusEvent.Status != api.ERX_STATUS_SENT {
			t.Fatalf("Expected status %s for top level status of unlinked treatment but got %s", api.ERX_STATUS_SENT, unlinkedTreatmentStatusEvent.Status)
		}
	}
}

func TestDenyRefillRequestWithDNTFUnlinkedTreatmentErrorSending(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	unlinkedTreatment := setUpDeniedRefillRequestWithDNTF(t, testData, api.ERX_STATUS_ERROR, false)

	if len(unlinkedTreatment.ERx.RxHistory) != 3 {
		t.Fatalf("Expcted 3 events from rx history of unlinked treatment instead got %d", len(unlinkedTreatment.ERx.RxHistory))
	}

	for _, unlinkedTreatmentStatusEvent := range unlinkedTreatment.ERx.RxHistory {
		if unlinkedTreatmentStatusEvent.InternalStatus == api.STATUS_ACTIVE && unlinkedTreatmentStatusEvent.Status != api.ERX_STATUS_ERROR {
			t.Fatalf("Expected status %s for top level status of unlinked treatment but got %s", api.ERX_STATUS_SENT, unlinkedTreatmentStatusEvent.Status)
		}
	}

	// check if this results in an item in the doctor queue
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(unlinkedTreatment.Doctor.DoctorId.Int64())
	if err != nil {
		t.Fatalf("Unable to get pending items for doctor: %+v", err)
	}

	if len(pendingItems) != 1 {
		t.Fatalf("Expected 1 pending item in the doctor queue instead got %d", len(pendingItems))
	}

	if pendingItems[0].EventType != api.EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR {
		t.Fatalf("Expected event type of item in doctor queue to be %s but was %s instead", api.EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR, pendingItems[0].EventType)
	}

	stubErxApi := &erx.StubErxService{}
	// lets go ahead and resolve the error, which should also clear the pending items from the doctor queue
	doctorPrescriptionErrorIgnoreHandler := &apiservice.DoctorPrescriptionErrorIgnoreHandler{
		DataApi: testData.DataApi,
		ErxApi:  stubErxApi,
	}

	params := &url.Values{}
	params.Set("unlinked_dntf_treatment_id", strconv.FormatInt(unlinkedTreatment.Id.Int64(), 10))

	ignoreErrorTs := httptest.NewServer(doctorPrescriptionErrorIgnoreHandler)
	defer ignoreErrorTs.Close()

	resp, err := authPost(ignoreErrorTs.URL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()), unlinkedTreatment.Doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to successfully resolve error pertaining to unlinked dntf treatment: %+v", err)
	}

	CheckSuccessfulStatusCode(resp, "Unable to successfully resolve error pertaining to unlinked dntf treatment", t)
}

func setUpDeniedRefillRequestWithDNTFForLinkedTreatment(t *testing.T, testData TestData, endErxStatus string, toAddTemplatedTreatment bool) *common.Treatment {
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

	comment := "this is a test"
	treatmentToAdd := common.Treatment{
		DrugInternalName: "Testing (If - This Works)",
		DrugDBIds: map[string]string{
			erx.LexiSynonymTypeId: "12345",
			erx.LexiDrugSynId:     "123151",
			erx.LexiGenProductId:  "124151",
			erx.NDC:               "1415",
		},
		DosageStrength:      "10 mg",
		DispenseValue:       1,
		DispenseUnitId:      common.NewObjectId(12),
		NumberRefills:       1,
		OTC:                 false,
		PatientInstructions: "patient instructions",
	}

	if toAddTemplatedTreatment {

		treatmentTemplate := &common.DoctorTreatmentTemplate{}
		treatmentTemplate.Name = "Favorite Treatment #1"
		treatmentTemplate.Treatment = &treatmentToAdd

		doctorFavoriteTreatmentsHandler := &apiservice.DoctorTreatmentTemplatesHandler{DataApi: testData.DataApi}
		ts := httptest.NewServer(doctorFavoriteTreatmentsHandler)
		defer ts.Close()

		treatmentTemplatesRequest := &apiservice.DoctorTreatmentTemplatesRequest{TreatmentTemplates: []*common.DoctorTreatmentTemplate{treatmentTemplate}}
		treatmentTemplatesRequest.PatientVisitId = common.NewObjectId(patientVisitResponse.PatientVisitId)
		data, err := json.Marshal(&treatmentTemplatesRequest)
		if err != nil {
			t.Fatal("Unable to marshal request body for adding treatments to patient visit")
		}

		resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(data), doctor.AccountId.Int64())
		if err != nil {
			t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Request to add treatments failed with http status code %d", resp.StatusCode)
		}

		treatmentTemplatesResponse := &apiservice.DoctorTreatmentTemplatesResponse{}
		err = json.NewDecoder(resp.Body).Decode(treatmentTemplatesResponse)
		if err != nil {
			t.Fatal("Unable to unmarshal response into object : " + err.Error())
		}

		treatmentToAdd.DoctorTreatmentTemplateId = treatmentTemplatesResponse.TreatmentTemplates[0].Id
	}

	testTime := time.Now()

	treatment1 := &common.Treatment{
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
		OTC:                     false,
		ERx: &common.ERxData{
			PrescriptionId:     common.NewObjectId(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      1234,
			PharmacyLocalId:    common.NewObjectId(pharmacyToReturn.LocalId),
			ErxLastDateFilled:  &testTime,
		},
	}

	// add this treatment to the treatment plan
	err = testData.DataApi.AddTreatmentsForPatientVisit([]*common.Treatment{treatment1}, doctor.DoctorId.Int64(), treatmentPlanId, signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to add treatment for patient visit: " + err.Error())
	}

	// insert erxStatusEvent for this treatment to indicate that it was sent
	_, err = testData.DB.Exec(`insert into erx_status_events (treatment_id, erx_status, creation_date, status) values (?,?,?,?)`, treatment1.Id.Int64(), api.ERX_STATUS_SENT, testTime, "ACTIVE")
	if err != nil {
		t.Fatal("Unable to insert erx_status_events x`")
	}

	// update the treatment with prescription id and pharmacy id for where prescription was routed
	_, err = testData.DB.Exec(`update treatment set erx_id = ?, pharmacy_id=? where id = ?`, treatment1.ERx.PrescriptionId.Int64(), pharmacyToReturn.LocalId, treatment1.Id.Int64())
	if err != nil {
		t.Fatal("Unable to update treatment with erx id: " + err.Error())
	}
	prescriptionIdForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemId := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemId:      refillRequestQueueItemId,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientId:              erxPatientId,
		PatientAddedForRequest:    false,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
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
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianId: clinicianId,
				PrescriptionId:      common.NewObjectId(prescriptionIdForRequestedPrescription),
				ErxPharmacyId:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       1234,
				DoseSpotClinicianId: clinicianId,
			},
		},
	}

	prescriptionIdForTreatment := int64(15515616)
	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
		PrescriptionIdsToReturn:      []int64{prescriptionIdForTreatment},
		PrescriptionIdToPrescriptionStatuses: map[int64][]common.StatusEvent{
			prescriptionIdForTreatment: []common.StatusEvent{common.StatusEvent{
				Status: endErxStatus,
			},
			},
		},
		SelectedMedicationToReturn: &common.Treatment{},
	}

	// Call the Consume method
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

	refillRequestStatuses, err := testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	denialReasons, err := testData.DataApi.GetRefillRequestDenialReasons()
	if err != nil || len(denialReasons) == 0 {
		t.Fatal("Unable to get the denial reasons for the refill request")
	}

	var dntfReason *api.RefillRequestDenialReason
	for _, denialReason := range denialReasons {
		if denialReason.DenialCode == api.RX_REFILL_DNTF_REASON_CODE {
			dntfReason = denialReason
			break
		}
	}

	if dntfReason == nil {
		t.Fatal("Unable to find DNTF reason in database: " + err.Error())
	}

	erxStatusQueue := &common.SQSQueue{}
	erxStatusQueue.QueueService = &sqs.StubSQS{}
	erxStatusQueue.QueueUrl = "local-erx"

	// now, lets go ahead and attempt to deny this refill request

	requestData := apiservice.DoctorRefillRequestRequestData{
		RefillRequestId: common.NewObjectId(refillRequest.Id),
		Action:          "deny",
		DenialReasonId:  common.NewObjectId(dntfReason.Id),
		Comments:        comment,
		Treatment:       &treatmentToAdd,
	}

	doctorRefillRequestsHandler := &apiservice.DoctorRefillRequestHandler{
		DataApi:        testData.DataApi,
		ErxApi:         stubErxAPI,
		ErxStatusQueue: erxStatusQueue,
	}

	// sleep for a brief moment before denyingh so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	ts := httptest.NewServer(doctorRefillRequestsHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		t.Fatal("Unable to marshal json into object: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to deny refill request: " + err.Error())
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful request to deny refill request: "+string(respBody), t)

	// get refill request to ensure that it was denied
	refillRequest, err = testData.DataApi.GetRefillRequestFromId(refillRequest.Id)
	if err != nil {
		t.Fatalf("Unable to get refill request from id: %+v", err)
	}

	if len(refillRequest.RxHistory) != 2 {
		t.Fatalf("Expected there to be 2 refill request events instead there were %d", len(refillRequest.RxHistory))
	}

	if refillRequest.RxHistory[0].Status != api.RX_REFILL_STATUS_DENIED {
		t.Fatalf("Expected top level refill request status of %s instead got %s", refillRequestItem.RxHistory[0].Status, api.RX_REFILL_STATUS_DENIED)
	}

	// get unlinked treatment
	unlinkedDNTFTreatmentStatusEvents, err := testData.DataApi.GetErxStatusEventsForDNTFTreatmentBasedOnPatientId(refillRequest.Patient.PatientId.Int64())
	if err != nil {
		t.Fatalf("Unable to get status events for dntf treatment: %+v", err)
	}

	if len(unlinkedDNTFTreatmentStatusEvents) != 0 {
		t.Fatalf("Expected 0 status events for unlinked dntf treatments instead got %d", len(unlinkedDNTFTreatmentStatusEvents))
	}

	// check dntf mapping to ensure that there is an entry
	var dntfMappingCount int64
	if err = testData.DB.QueryRow(`select count(*) from dntf_mapping`).Scan(&dntfMappingCount); err != nil {
		t.Fatalf("Unable to count number of entries in dntf mapping table: %+v", err)
	}

	if dntfMappingCount != 1 {
		t.Fatalf("Expected 1 entry in dntf mapping table instead got %d", dntfMappingCount)
	}

	treatments, err := testData.DataApi.GetTreatmentsBasedOnTreatmentPlanId(patientVisitResponse.PatientVisitId, treatmentPlanId)
	if err != nil {
		t.Fatalf("Unable to get the treatmend based on prescription id: %+v", err)
	}

	if len(treatments) != 2 {
		t.Fatalf("Expected 2 treatments in treatment plan instead got %d", len(treatments))
	}

	var linkedTreatment *common.Treatment
	for _, treatment := range treatments {
		if treatment.ERx.PrescriptionId.Int64() == prescriptionIdForTreatment {
			linkedTreatment = treatment
			break
		}
	}

	if linkedTreatment == nil {
		t.Fatalf("Unable to find the treatment that was added as a result of DNTF")
	}

	if toAddTemplatedTreatment {
		if linkedTreatment.DoctorTreatmentTemplateId == nil || linkedTreatment.DoctorTreatmentTemplateId.Int64() == 0 {
			t.Fatal("Expected there to exist a doctor template id given that the treatment was created from a template but there wasnt one")
		}
	}

	// the treatment as a result of DNTF, if linked, should map back to the original treatemtn plan
	// associated with the originating treatment for the refill request
	if linkedTreatment.TreatmentPlanId == nil || linkedTreatment.TreatmentPlanId.Int64() != treatmentPlanId {
		t.Fatalf("Expected the linked treatment to map back to the original treatment but it didnt")
	}

	if len(linkedTreatment.ERx.RxHistory) != 2 {
		t.Fatalf("Expected there to be 2 events for this linked dntf treatment, instead got %d", len(linkedTreatment.ERx.RxHistory))
	}

	for _, linkedTreatmentStatus := range linkedTreatment.ERx.RxHistory {
		if linkedTreatmentStatus.InternalStatus == api.STATUS_INACTIVE && linkedTreatmentStatus.Status != api.ERX_STATUS_NEW_RX_FROM_DNTF {
			t.Fatalf("Expected the first event for the linked treatment to be %s instead it was %s", api.ERX_STATUS_NEW_RX_FROM_DNTF, linkedTreatmentStatus.Status)
		}
	}

	// check erx status to be sent once its sent
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	linkedTreatment, err = testData.DataApi.GetTreatmentBasedOnPrescriptionId(prescriptionIdForTreatment)
	if err != nil {
		t.Fatalf("Unable to get the treatmend based on prescription id: %+v", err)
	}
	return linkedTreatment
}

func TestDenyRefillRequestWithDNTFWithLinkedTreatmentSuccessfulSend(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	linkedTreatment := setUpDeniedRefillRequestWithDNTFForLinkedTreatment(t, testData, api.ERX_STATUS_SENT, false)

	if len(linkedTreatment.ERx.RxHistory) != 3 {
		t.Fatalf("Expected 3 events for linked treatment instead got %d", len(linkedTreatment.ERx.RxHistory))
	}

	for _, linkedTreatmentStatusEvent := range linkedTreatment.ERx.RxHistory {
		if linkedTreatmentStatusEvent.InternalStatus == api.STATUS_ACTIVE && linkedTreatmentStatusEvent.Status != api.ERX_STATUS_SENT {
			t.Fatalf("Expected the latest event for the linked treatment to be %s instead it was %s", api.ERX_STATUS_SENT, linkedTreatmentStatusEvent.Status)
		}
	}
}

func TestDenyRefillRequestWithDNTFWithLinkedTreatmentSuccessfulSendAddingFromTemplatedTreatment(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	linkedTreatment := setUpDeniedRefillRequestWithDNTFForLinkedTreatment(t, testData, api.ERX_STATUS_SENT, true)

	if len(linkedTreatment.ERx.RxHistory) != 3 {
		t.Fatalf("Expected 3 events for linked treatment instead got %d", len(linkedTreatment.ERx.RxHistory))
	}

	for _, linkedTreatmentStatusEvent := range linkedTreatment.ERx.RxHistory {
		if linkedTreatmentStatusEvent.InternalStatus == api.STATUS_ACTIVE && linkedTreatmentStatusEvent.Status != api.ERX_STATUS_SENT {
			t.Fatalf("Expected the latest event for the linked treatment to be %s instead it was %s", api.ERX_STATUS_SENT, linkedTreatmentStatusEvent.Status)
		}
	}
}

func TestDenyRefillRequestWithDNTFWithLinkedTreatmentErrorSend(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	linkedTreatment := setUpDeniedRefillRequestWithDNTFForLinkedTreatment(t, testData, api.ERX_STATUS_ERROR, false)

	if len(linkedTreatment.ERx.RxHistory) != 3 {
		t.Fatalf("Expected 3 events for linked treatment instead got %d", len(linkedTreatment.ERx.RxHistory))
	}

	for _, linkedTreatmentStatusEvent := range linkedTreatment.ERx.RxHistory {
		if linkedTreatmentStatusEvent.InternalStatus == api.STATUS_ACTIVE && linkedTreatmentStatusEvent.Status != api.ERX_STATUS_ERROR {
			t.Fatalf("Expected the latest event for the linked treatment to be %s instead it was %s", api.ERX_STATUS_ERROR, linkedTreatmentStatusEvent.Status)
		}
	}

	// there should be one item in the doctor's queue relating to a transmission error
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(linkedTreatment.Doctor.DoctorId.Int64())
	if err != nil {
		t.Fatalf("Unable to get pending items from doctors queue: %+v", err)
	}

	if len(pendingItems) != 1 {
		t.Fatalf("Expected there to be 1 item in the doctors queue instead there were %d", len(pendingItems))
	}

	if pendingItems[0].EventType != api.EVENT_TYPE_TRANSMISSION_ERROR {
		t.Fatalf("Expected the one item in the doctors queue to be of type %s instead it was of type %s", api.EVENT_TYPE_TRANSMISSION_ERROR, pendingItems[0].EventType)
	}
}

func TestCheckingStatusOfMultipleRefillRequestsAtOnce(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianId(testData, t)

	approvedRefillRequestPrescriptionId := int64(101010)
	approvedRefillAmount := int64(10)

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

	err := testData.DataApi.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIdForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemId := int64(12345)
	refillRequests := make([]*common.RefillRequestItem, 0)
	for i := int64(0); i < 4; i++ {
		// Get StubErx to return refill requests in the refillRequest call
		refillRequestItem := &common.RefillRequestItem{
			RxRequestQueueItemId:      refillRequestQueueItemId + i,
			ReferenceNumber:           "TestReferenceNumber",
			PharmacyRxReferenceNumber: "TestRxReferenceNumber",
			ErxPatientId:              12345,
			PatientAddedForRequest:    true,
			RequestDateStamp:          testTime,
			ClinicianId:               clinicianId,
			RequestedPrescription: &common.Treatment{
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
				ERx: &common.ERxData{
					ErxSentDate:         &fiveMinutesBeforeTestTime,
					DoseSpotClinicianId: clinicianId,
					PrescriptionId:      common.NewObjectId(prescriptionIdForRequestedPrescription + i),
					ErxPharmacyId:       1234,
				},
			},
			DispensedPrescription: &common.Treatment{
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
				OTC:                     false,
				ERx: &common.ERxData{
					DoseSpotClinicianId: clinicianId,
					PrescriptionId:      common.NewObjectId(5504),
					PrescriptionStatus:  "Requested",
					ErxPharmacyId:       1234,
					ErxLastDateFilled:   &testTime,
				},
			},
		}
		refillRequests = append(refillRequests, refillRequestItem)
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		PatientDetailsToReturn:       patientToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequests[0]},
		RefillRequestPrescriptionIds: map[int64]int64{
			refillRequestQueueItemId: approvedRefillRequestPrescriptionId,
		},
		PrescriptionIdToPrescriptionStatuses: map[int64][]common.StatusEvent{
			approvedRefillRequestPrescriptionId: []common.StatusEvent{common.StatusEvent{
				Status: api.ERX_STATUS_SENT,
			},
			},
		},
	}

	// Call the Consume method so that the first refill request gets added to the system
	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")

	refillRequestStatuses, err := testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	// lets go ahead and approve this refill request
	comment := "this is a test"
	requestData := apiservice.DoctorRefillRequestRequestData{
		RefillRequestId:      common.NewObjectId(refillRequest.Id),
		Action:               "approve",
		ApprovedRefillAmount: approvedRefillAmount,
		Comments:             comment,
	}

	erxStatusQueue := &common.SQSQueue{}

	stubSqs := &sqs.StubSQS{}
	erxStatusQueue.QueueService = stubSqs
	erxStatusQueue.QueueUrl = "local-erx"

	doctorRefillRequestsHandler := &apiservice.DoctorRefillRequestHandler{
		DataApi:        testData.DataApi,
		ErxApi:         stubErxAPI,
		ErxStatusQueue: erxStatusQueue,
	}

	// sleep for a brief moment before approving so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	ts := httptest.NewServer(doctorRefillRequestsHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Unable to marshal json object: %+v", err)
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successful request to approve refill request: ")
	}

	refillRequest, err = testData.DataApi.GetRefillRequestFromId(refillRequest.Id)
	if err != nil {
		t.Fatal("Unable to get refill request after approving request: " + err.Error())
	}

	// now lets go ahead and ensure that the refill request is successfully sent to the pharmacy
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	// now, lets go ahead and get 3 refill requests queued up for the clinic
	stubErxAPI.RefillRxRequestQueueToReturn = refillRequests[1:]
	stubErxAPI.RefillRequestPrescriptionIds = map[int64]int64{
		refillRequestQueueItemId:     approvedRefillRequestPrescriptionId,
		refillRequestQueueItemId + 1: approvedRefillRequestPrescriptionId + 1,
		refillRequestQueueItemId + 2: approvedRefillRequestPrescriptionId + 2,
		refillRequestQueueItemId + 3: approvedRefillRequestPrescriptionId + 3,
	}
	stubErxAPI.PrescriptionIdToPrescriptionStatuses = map[int64][]common.StatusEvent{
		approvedRefillRequestPrescriptionId: []common.StatusEvent{common.StatusEvent{
			Status: api.ERX_STATUS_SENT,
		},
		},
		approvedRefillRequestPrescriptionId + 1: []common.StatusEvent{common.StatusEvent{
			Status: api.ERX_STATUS_SENT,
		},
		},
		approvedRefillRequestPrescriptionId + 2: []common.StatusEvent{common.StatusEvent{
			Status: api.ERX_STATUS_SENT,
		},
		},
		approvedRefillRequestPrescriptionId + 3: []common.StatusEvent{common.StatusEvent{
			Status: api.ERX_STATUS_SENT,
		},
		},
	}

	app_worker.PerformRefillRecquestCheckCycle(testData.DataApi, stubErxAPI, metrics.NewCounter(), metrics.NewCounter(), "test")
	refillRequestStatuses, err = testData.DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	if len(refillRequestStatuses) != 3 {
		t.Fatalf("Expected 3 refill requests to be queued up in the REQUESTED state, instead we have %d", len(refillRequestStatuses))
	}

	// now lets go ahead and approve all of the refill requests
	// sleep for a brief moment before approving so that
	// the items are ordered correctly for the rx history (in the real world they would not be approved in the same exact millisecond they are sent in)
	time.Sleep(1 * time.Second)

	// go ahead and approve all remaining refill requests
	for i := 0; i < len(refillRequestStatuses); i++ {

		requestData.RefillRequestId = common.NewObjectId(refillRequestStatuses[i].ItemId)
		jsonData, err = json.Marshal(&requestData)
		if err != nil {
			t.Fatalf("Unable to marshal json object: %+v", err)
		}

		resp, err = authPut(ts.URL, "application/x-www-form-urlencoded", bytes.NewReader(jsonData), doctor.AccountId.Int64())
		if err != nil {
			t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatal("Unable to make successful request to approve refill request: ")
		}
	}

	// now lets go ahead and ensure that the refill request is successfully sent to the pharmacy
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	// all 3 refill requests should not have 3 items in the rx history
	refillRequestStatusEvents, err := testData.DataApi.GetRefillStatusEventsForRefillRequest(refillRequestStatuses[0].ItemId)
	if err != nil {
		t.Fatal("Error while trying to get refill request status events: " + err.Error())
	}

	if len(refillRequestStatusEvents) != 3 {
		t.Fatalf("Expected 3 refill request events instead got %d", len(refillRequestStatusEvents))
	}

	refillRequestStatusEvents, err = testData.DataApi.GetRefillStatusEventsForRefillRequest(refillRequestStatuses[1].ItemId)
	if err != nil {
		t.Fatal("Error while trying to get refill request status events: " + err.Error())
	}

	if len(refillRequestStatusEvents) != 3 {
		t.Fatalf("Expected 3 refill request events instead got %d", len(refillRequestStatusEvents))
	}

	refillRequestStatusEvents, err = testData.DataApi.GetRefillStatusEventsForRefillRequest(refillRequestStatuses[2].ItemId)
	if err != nil {
		t.Fatal("Error while trying to get refill request status events: " + err.Error())
	}

	if len(refillRequestStatusEvents) != 3 {
		t.Fatalf("Expected 3 refill request events instead got %d", len(refillRequestStatusEvents))
	}

	if stubSqs.MsgQueue[erxStatusQueue.QueueUrl].Len() != 2 {
		t.Fatalf("Expected 2 items to remain in the msg queue instead got %d", len(stubSqs.MsgQueue))
	}

	// now lets go ahead and ensure that the refill request is successfully sent to the pharmacy
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	if stubSqs.MsgQueue[erxStatusQueue.QueueUrl].Len() != 1 {
		t.Fatalf("Expected 1 item to remain in the msg queue instead got %d", len(stubSqs.MsgQueue))
	}

	// now lets go ahead and ensure that the refill request is successfully sent to the pharmacy
	app_worker.ConsumeMessageFromQueue(testData.DataApi, stubErxAPI, erxStatusQueue, metrics.NewBiasedHistogram(), metrics.NewCounter(), metrics.NewCounter())

	if stubSqs.MsgQueue[erxStatusQueue.QueueUrl].Len() != 0 {
		t.Fatalf("Expected 0 item to remain in the msg queue instead got %d", len(stubSqs.MsgQueue))
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
		OTC:                     false,
		ERx: &common.ERxData{
			ErxLastDateFilled:  &testTime,
			PrescriptionId:     common.NewObjectId(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyId:      1234,
			PharmacyLocalId:    common.NewObjectId(pharmacyToReturn.LocalId),
		},
	}

	// add this treatment to the treatment plan
	err = testData.DataApi.AddTreatmentsForPatientVisit([]*common.Treatment{treatment1}, doctor.DoctorId.Int64(), treatmentPlanId, signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to add treatment for patient visit: " + err.Error())
	}

	// insert erxStatusEvent for this treatment to indicate that it was sent
	_, err = testData.DB.Exec(`insert into erx_status_events (treatment_id, erx_status, creation_date, status) values (?,?,?,?)`, treatment1.Id.Int64(), api.ERX_STATUS_SENT, testTime, "ACTIVE")
	if err != nil {
		t.Fatal("Unable to insert erx_status_events x`")
	}

	// update the treatment with prescription id and pharmacy id for where prescription was routed
	_, err = testData.DB.Exec(`update treatment set erx_id = ?, pharmacy_id=? where id = ?`, treatment1.ERx.PrescriptionId.Int64(), pharmacyToReturn.LocalId, treatment1.Id.Int64())
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
		ErxPatientId:              erxPatientId,
		PatientAddedForRequest:    false,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
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
			ERx: &common.ERxData{
				DoseSpotClinicianId: clinicianId,
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				PrescriptionId:      common.NewObjectId(prescriptionIdForRequestedPrescription),
				ErxPharmacyId:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       12345678,
				DoseSpotClinicianId: clinicianId,
			},
		},
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
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

	if refillRequestStatuses[0].ItemId != refillRequestItem.Id ||
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
		pendingItems[0].ItemId != refillRequestStatuses[0].ItemId {
		t.Fatal("Pending item found in the doctor's queue is not the expected item")
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
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

	if refillRequest.RequestedPrescription.ERx.Pharmacy == nil || refillRequest.DispensedPrescription.ERx.Pharmacy == nil {
		t.Fatal("Expected pharmacy object to be present for requested and dispensed prescriptions")
	}

	if refillRequest.RequestedPrescription.ERx.Pharmacy.SourceId == refillRequest.DispensedPrescription.ERx.Pharmacy.SourceId {
		t.Fatal("Expected the pharmacies to be different between the requested and the dispensed prescriptions")
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
		ErxPatientId:              erxPatientId,
		PatientAddedForRequest:    false,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				DoseSpotClinicianId: clinicianId,
				ErxSentDate:         &testTime,
				PrescriptionId:      common.NewObjectId(prescriptionIdForRequestedPrescription),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       123,
			},
		},
		DispensedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       123,
				ErxSentDate:         &testTime,
				DoseSpotClinicianId: clinicianId,
			},
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

	if refillRequestStatuses[0].ItemId != refillRequestItem.Id ||
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
		pendingItems[0].ItemId != refillRequestStatuses[0].ItemId {
		t.Fatal("Pending item found in the doctor's queue is not the expected item")
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
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
		ErxPatientId:              555,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
		ClinicianId:               clinicianId,
		RequestedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				DoseSpotClinicianId: clinicianId,
				ErxSentDate:         &testTime,
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       123,
			},
		},
		DispensedPrescription: &common.Treatment{
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
			OTC:                     false,
			ERx: &common.ERxData{
				DoseSpotClinicianId: clinicianId,
				PrescriptionId:      common.NewObjectId(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyId:       123,
				ErxSentDate:         &testTime,
			},
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

	if refillRequestStatuses[0].ItemId != refillRequestItem.Id ||
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
		pendingItems[0].ItemId != refillRequestStatuses[0].ItemId {
		t.Fatal("Pending item found in the doctor's queue is not the expected item")
	}

	refillRequest, err := testData.DataApi.GetRefillRequestFromId(refillRequestStatuses[0].ItemId)
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
