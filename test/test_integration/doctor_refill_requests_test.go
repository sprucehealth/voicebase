package test_integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	doctorpkg "github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/test"
)

const (
	clinicianID = 100
)

// TestRefill_ExistingPatient is an integration test
// for refill requests coming in for patients that exist on the Spruce platform.
func TestRefill_ExistingPatient(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)

	doctorClient := DoctorClient(testData, t, doctor.ID.Int64())

	signedupPatientResponse := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	erxPatientID := int64(60)

	// add an erx patient id to the patient
	err := testData.DataAPI.UpdatePatient(signedupPatientResponse.Patient.ID.Int64(),
		&api.PatientUpdate{
			ERxID: &erxPatientID,
		}, false)
	if err != nil {
		t.Fatal("Unable to update patient with erx patient id : " + err.Error())
	}

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err = testData.DataAPI.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	pv, _ := CreatePatientVisitAndPickTP(t, testData, signedupPatientResponse.Patient, doctor)

	// start a new treatemtn plan for the patient visit
	tp, err := doctorClient.PickTreatmentPlanForVisit(pv.PatientVisitID, nil)
	test.OK(t, err)
	treatmentPlanID := tp.ID.Int64()

	testTime := time.Now()

	treatment1 := &common.Treatment{
		DrugDBIDs: map[string]string{
			erx.LexiDrugSynID:     "1234",
			erx.LexiGenProductID:  "12345",
			erx.LexiSynonymTypeID: "123556",
			erx.NDC:               "2415",
		},
		DrugName:                "Testing",
		DrugRoute:               "topical",
		DrugForm:                "cream",
		DosageStrength:          "10 mg",
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

		ERx: &common.ERxData{
			PrescriptionID:     encoding.NewObjectID(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyID:      1234,
			PharmacyLocalID:    encoding.NewObjectID(pharmacyToReturn.LocalID),
			ErxLastDateFilled:  &testTime,
		},
	}

	// add this treatment to the treatment plan
	err = testData.DataAPI.AddTreatmentsForTreatmentPlan([]*common.Treatment{treatment1}, doctor.ID.Int64(), treatmentPlanID, signedupPatientResponse.Patient.ID.Int64())
	if err != nil {
		t.Fatal("Unable to add treatment for patient visit: " + err.Error())
	}

	// insert erxStatusEvent for this treatment to indicate that it was sent
	_, err = testData.DB.Exec(`insert into erx_status_events (treatment_id, erx_status, creation_date, status) values (?,?,?,?)`, treatment1.ID.Int64(), api.ERXStatusSent, testTime, "ACTIVE")
	if err != nil {
		t.Fatal("Unable to insert erx_status_events x`")
	}

	// update the treatment with prescription id and pharmacy id for where prescription was routed
	_, err = testData.DB.Exec(`update treatment set erx_id = ?, pharmacy_id=? where id = ?`, treatment1.ERx.PrescriptionID.Int64(), pharmacyToReturn.LocalID, treatment1.ID.Int64())
	if err != nil {
		t.Fatal("Unable to update treatment with erx id: " + err.Error())
	}
	prescriptionIDForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemID := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      refillRequestQueueItemID,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              erxPatientID,
		PatientAddedForRequest:    false,
		RequestDateStamp:          testTime,
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
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription),
				ErxPharmacyID:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				"drug_db_id_1": "1234",
				"drug_db_id_2": "12345",
			},
			DrugName:                "Testing",
			DrugRoute:               "topical",
			DrugForm:                "cream",
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
				ErxLastDateFilled:   &testTime,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       1234,
				DoseSpotClinicianID: clinicianID,
			},
		},
	}

	stubErxAPI := &erx.StubErxService{
		PharmacyDetailsToReturn:      pharmacyToReturn,
		RefillRxRequestQueueToReturn: []*common.RefillRequestItem{refillRequestItem},
		PrescriptionIDToPrescriptionStatuses: map[int64][]common.StatusEvent{
			prescriptionIDForRequestedPrescription: []common.StatusEvent{{
				Status: api.ERXStatusSent,
			}},
		},
	}

	// Call the Consume method
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	refillRXWorker.Do()

	var count int64
	err = testData.DB.QueryRow(`select count(*) from requested_treatment`).Scan(&count)
	if err != nil {
		t.Fatal("Unable to get a count for the unumber of treatments in the requested_treatment table " + err.Error())
	}
	if count == 0 {
		t.Fatalf("Expected there to be a requested treatment, but got none")
	}

	// There should be a status entry in the refill_request_status table
	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	if len(refillRequestStatuses) != 1 {
		t.Fatal("Expected there to exist 1 refill request status for the refill request just persisted")
	}

	if refillRequestStatuses[0].ItemID != refillRequestItem.ID ||
		refillRequestStatuses[0].Status != api.RXRefillStatusRequested {
		t.Fatal("Refill request status not in expected state")
	}

	// There should be a pending entry in the doctor's queue
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	if err != nil {
		t.Fatal("Unable to get pending items from doctor queue: " + err.Error())
	}

	if len(pendingItems) != 2 {
		t.Fatalf("Expected 2 pending items in the doctor queue but got %d", len(pendingItems))
	} else if pendingItems[1].EventType != api.DQEventTypeRefillRequest {
		t.Fatalf("Expected doctor queue item type of %s but got %s", pendingItems[0].EventType, api.DQEventTypeRefillRequest)
	} else if pendingItems[1].ItemID != refillRequestStatuses[0].ItemID {
		t.Fatalf("Refill request item does not have expected id. Expected %d got %d", pendingItems[1].ItemID, refillRequestStatuses[0].ItemID)
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	if refillRequest.DispensedPrescription == nil {
		t.Fatalf("Dispensed prescription was null for the refill request when it shouldn't be")
	}

	if refillRequest.RequestedPrescription == nil {
		t.Fatal("Requested prescription was null for refill request when it shouldn't be")
	}

	if refillRequest.RequestedPrescription.OriginatingTreatmentID == 0 {
		t.Fatal("Requested prescription should be one that was found in our system, but instead its indicated to be unlinked")
	}

	if !refillRequest.TreatmentPlanID.IsValid || refillRequest.TreatmentPlanID.Int64() == 0 {
		t.Fatal("Expected treatment plan id to be set given that the treatment is linked")
	}

	if refillRequest.Patient == nil {
		t.Fatal("Refill request expected to have patient demographics attached to it instead it doesnt")
	}

	if refillRequest.Patient.Status != api.PatientRegistered {
		t.Fatal("Patient requesting refill expected to be in our system instead the indication is that it was an unlinked patient")
	}
}

// TestRefill_Approve is an integration test to test the approving of a refill request
// for a new patient.
func TestRefill_Approve(t *testing.T) {

	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)

	approvedRefillRequestPrescriptionID := int64(101010)
	approvedRefillAmount := int64(10)

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
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
		DOB:          encoding.Date{Day: 11, Month: 11, Year: 1980},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(12345),
	}

	err := testData.DataAPI.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIDForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemID := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      refillRequestQueueItemID,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
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
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription),
				ErxPharmacyID:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				"drug_db_id_1": "1234",
				"drug_db_id_2": "12345",
			},
			DrugName:                "Testing",
			DrugRoute:               "topical",
			DrugForm:                "cream",
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
				Int64Value: 5,
			}, PatientInstructions: "Take once daily",
			OTC: false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       1234,
			},
		},
	}

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}
	stubErxAPI.RefillRequestPrescriptionIDs = map[int64]int64{
		refillRequestQueueItemID: approvedRefillRequestPrescriptionID,
	}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		approvedRefillRequestPrescriptionID: []common.StatusEvent{common.StatusEvent{
			Status: api.ERXStatusSent,
		},
		},
	}

	// Call the Consume method
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	refillRXWorker.Do()

	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	// lets go ahead and attempt to approve this refill request
	comment := "this is a test"

	approveRefillRequest(refillRequest, doctor.AccountID.Int64(), comment, testData, t)
	refillRequest, err = testData.DataAPI.GetRefillRequestFromID(refillRequest.ID)
	if err != nil {
		t.Fatal("Unable to get refill request after approving request: " + err.Error())
	}

	if len(refillRequest.RxHistory) != 2 {
		t.Fatalf("Expected 2 items in the rx history for the refill request instead got %d", len(refillRequest.RxHistory))
	}

	if refillRequest.RxHistory[0].Status != api.RXRefillStatusApproved {
		t.Fatalf("Expected the refill request status to be %s but was %s instead: %+v", api.RXRefillStatusApproved, refillRequest.RxHistory[0].Status, refillRequest.RxHistory)
	}

	if refillRequest.ApprovedRefillAmount != approvedRefillAmount {
		t.Fatalf("Expected the approved refill amount to be %d but wsa %d instead", approvedRefillRequestPrescriptionID, refillRequest.ApprovedRefillAmount)
	}

	if refillRequest.Comments != comment {
		t.Fatalf("Expected the comment on the refill request to be '%s' but was '%s' instead", comment, refillRequest.Comments)
	}

	if refillRequest.PrescriptionID != approvedRefillRequestPrescriptionID {
		t.Fatalf("Expected the prescription id returned to be %d but instead it was %d", approvedRefillAmount, refillRequest.PrescriptionID)
	}

	// doctor queue should be empty and the approved request should be in the completed tab
	completedItems, err := testData.DataAPI.GetCompletedItemsInDoctorQueue(doctor.ID.Int64())
	if err != nil {
		t.Fatal("Unable to get the completed items for the doctor: " + err.Error())
	}

	if len(completedItems) != 1 {
		t.Fatal("Expected there to be 1 completed item in the doctor's queue for the refill request that was just rejected")
	}

	if completedItems[0].EventType != api.DQEventTypeRefillRequest || completedItems[0].ItemID != refillRequest.ID ||
		completedItems[0].Status != api.DQItemStatusRefillApproved {
		t.Fatal("Completed item in the doctor's queue not in the expected state")
	}

	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	if err != nil {
		t.Fatal("Unable to get the pending items for the doctor: " + err.Error())
		return
	}

	if len(pendingItems) != 0 {
		t.Fatalf("Expected there to be no pending items in the doctor's queue instead there were %d", len(pendingItems))
	}

	// attempt to consume the message put into the queue
	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	for {
		consumed, err := statusWorker.Do()
		test.OK(t, err)
		if !consumed {
			break
		}
	}

	// now, the status of the refill request should be Sent
	refillStatusEvents, err := testData.DataAPI.GetRefillStatusEventsForRefillRequest(refillRequest.ID)
	if err != nil {
		t.Fatal("Unable to get refill status events for refill request: " + err.Error())
	}

	if len(refillStatusEvents) != 3 {
		t.Fatalf("Expected 3 refill status events for refill request but got %d", len(refillStatusEvents))
	}

	if refillStatusEvents[0].Status != api.RXRefillStatusSent {
		t.Fatalf("Expected the top level item for the refill request to indicate that it was successfully sent to the pharmacy %+v", refillStatusEvents)
	}
}

// TestRefill_Approve_ControlledSubstance is an integration test to ensure
// that the system does not allow approving of refill requests for controlled substances.
func TestRefill_Approve_ControlledSubstance(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)

	approvedRefillRequestPrescriptionID := int64(101010)

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
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
		DOB:          encoding.Date{Day: 11, Month: 11, Year: 1980},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(12345),
	}

	err := testData.DataAPI.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIDForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemID := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      refillRequestQueueItemID,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
		ClinicianID:               clinicianID,
		RequestedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				erx.LexiDrugSynID:     "1234",
				erx.LexiGenProductID:  "12345",
				erx.LexiSynonymTypeID: "123556",
				erx.NDC:               "2415",
			},
			DosageStrength:        "10 mg",
			DispenseValue:         5,
			OTC:                   false,
			SubstitutionsAllowed:  true,
			IsControlledSubstance: true,
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription),
				ErxPharmacyID:       1234,
			},
		},
		DispensedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				"drug_db_id_1": "1234",
				"drug_db_id_2": "12345",
			},
			DrugName:                "Testing",
			DrugRoute:               "topical",
			DrugForm:                "cream",
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
				Int64Value: 5,
			}, PatientInstructions: "Take once daily",
			OTC: false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       1234,
			},
		},
	}

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}
	stubErxAPI.RefillRequestPrescriptionIDs = map[int64]int64{
		refillRequestQueueItemID: approvedRefillRequestPrescriptionID,
	}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		approvedRefillRequestPrescriptionID: []common.StatusEvent{common.StatusEvent{
			Status:        api.ERXStatusError,
			StatusDetails: "testing this error",
		},
		},
	}

	// Call the Consume method
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	refillRXWorker.Do()

	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	// lets go ahead and attempt to approve this refill request
	comment := "this is a test"
	requestData := doctorpkg.DoctorRefillRequestRequestData{
		RefillRequestID:      refillRequest.ID,
		Action:               "approve",
		ApprovedRefillAmount: 10,
		Comments:             comment,
	}

	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatal("Unable to marshal json object: " + err.Error())
	}

	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.DoctorRefillRxURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountID.Int64())
	if err != nil {
		t.Fatalf("Unable to make successful request to approve refill request: " + err.Error())
	} else if resp.StatusCode != apiservice.StatusUnprocessableEntity {
		t.Fatalf("Expected response code %d instead got %d", apiservice.StatusUnprocessableEntity, resp.StatusCode)
	}
	defer resp.Body.Close()

}

// TestRefill_Approve_ErrorSending is an integration test to ensure that if a prescriber
// approves a refill request that errors on the routing to a pharmacy, we gracefully handle this situation
// by inserting an errored prescription entry into the doctor queue.
func TestRefill_Approve_ErrorSending(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)
	doctorCli := DoctorClient(testData, t, doctor.ID.Int64())

	approvedRefillRequestPrescriptionID := int64(101010)
	approvedRefillAmount := int64(10)

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
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
		DOB:          encoding.Date{Year: 1987, Month: 1, Day: 22},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(12345),
	}

	err := testData.DataAPI.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIDForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemID := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      refillRequestQueueItemID,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
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
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription),
				ErxPharmacyID:       1234,
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
				Int64Value: 5,
			}, PatientInstructions: "Take once daily",

			OTC: false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       1234,
			},
		},
	}

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}
	stubErxAPI.RefillRequestPrescriptionIDs = map[int64]int64{
		refillRequestQueueItemID: approvedRefillRequestPrescriptionID,
	}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		approvedRefillRequestPrescriptionID: []common.StatusEvent{{
			Status:        api.ERXStatusError,
			StatusDetails: "testing this error",
		}},
	}

	// Call the Consume method
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	test.OK(t, refillRXWorker.Do())

	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	// lets go ahead and attempt to approve this refill request
	comment := "this is a test"

	approveRefillRequest(refillRequest, doctor.AccountID.Int64(), comment, testData, t)

	refillRequest, err = testData.DataAPI.GetRefillRequestFromID(refillRequest.ID)
	if err != nil {
		t.Fatal("Unable to get refill request after approving request: " + err.Error())
	}

	if len(refillRequest.RxHistory) != 2 {
		t.Fatalf("Expected 2 items in the rx history for the refill request instead got %d", len(refillRequest.RxHistory))
	}

	if refillRequest.RxHistory[0].Status != api.RXRefillStatusApproved {
		t.Fatalf("Expected the refill request status to be %s but was %s instead: %+v", api.RXRefillStatusApproved, refillRequest.RxHistory[0].Status, refillRequest.RxHistory)
	}

	if refillRequest.ApprovedRefillAmount != approvedRefillAmount {
		t.Fatalf("Expected the approved refill amount to be %d but wsa %d instead", approvedRefillAmount, refillRequest.ApprovedRefillAmount)
	}

	if refillRequest.Comments != comment {
		t.Fatalf("Expected the comment on the refill request to be '%s' but was '%s' instead", comment, refillRequest.Comments)
	}

	if refillRequest.PrescriptionID != approvedRefillRequestPrescriptionID {
		t.Fatalf("Expected the prescription id returned to be %d but instead it was %d", approvedRefillRequestPrescriptionID, refillRequest.PrescriptionID)
	}

	// doctor queue should be empty and the approved request should be in the completed tab
	completedItems, err := testData.DataAPI.GetCompletedItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(completedItems))
	if completedItems[0].EventType != api.DQEventTypeRefillRequest || completedItems[0].ItemID != refillRequest.ID ||
		completedItems[0].Status != api.DQItemStatusRefillApproved {
		t.Fatal("Completed item in the doctor's queue not in the expected state")
	}

	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 0, len(pendingItems))

	// attempt to consume the message put into the queue
	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	for {
		consumed, err := statusWorker.Do()
		test.OK(t, err)
		if !consumed {
			break
		}
	}

	refillStatusEvents, err := testData.DataAPI.GetRefillStatusEventsForRefillRequest(refillRequest.ID)
	test.OK(t, err)
	test.Equals(t, 3, len(refillStatusEvents))
	test.Equals(t, api.RXRefillStatusError, refillStatusEvents[0].Status)
	test.Equals(t, false, "" == refillStatusEvents[0].StatusDetails)

	// lets make sure that the error for the refill request makes it into the doctor's queue
	pendingItems, err = testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
	test.Equals(t, api.DQEventTypeRefillTransmissionError, pendingItems[0].EventType)

	// lets go ahead and resolve this error
	test.OK(t, doctorCli.ResolveRXErrorByRefillRequestID(refillRequest.ID))

	// check the rx history of the refill request
	refillRequest, err = testData.DataAPI.GetRefillRequestFromID(refillRequest.ID)
	test.OK(t, err)
	test.Equals(t, 4, len(refillRequest.RxHistory))
	test.Equals(t, api.RXRefillStatusErrorResolved, refillRequest.RxHistory[0].Status)

	pendingItems, err = testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 0, len(pendingItems))
}

func testRefill_Deny(isControlledSubstance bool, t *testing.T) {

	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)

	deniedRefillRequestPrescriptionID := int64(101010)

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
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
		DOB:          encoding.Date{Year: 1921, Month: 8, Day: 12},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(12345),
	}

	err := testData.DataAPI.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIDForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemID := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      refillRequestQueueItemID,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
		ClinicianID:               clinicianID,
		RequestedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				erx.LexiDrugSynID:     "1234",
				erx.LexiGenProductID:  "12345",
				erx.LexiSynonymTypeID: "123556",
				erx.NDC:               "2415",
			},
			DosageStrength:        "10 mg",
			DispenseValue:         5,
			IsControlledSubstance: isControlledSubstance,
			DaysSupply:            encoding.NullInt64{},
			OTC:                   false,
			SubstitutionsAllowed:  true,
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription),
				ErxPharmacyID:       1234,
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
				Int64Value: 5,
			}, PatientInstructions: "Take once daily",
			OTC: false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       1234,
				DoseSpotClinicianID: clinicianID,
			},
		},
	}

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}
	stubErxAPI.RefillRequestPrescriptionIDs = map[int64]int64{
		refillRequestQueueItemID: deniedRefillRequestPrescriptionID,
	}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		deniedRefillRequestPrescriptionID: []common.StatusEvent{common.StatusEvent{
			Status: api.ERXStatusDeleted,
		},
		},
	}

	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	refillRXWorker.Do()

	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	denialReasons, err := testData.DataAPI.GetRefillRequestDenialReasons()
	if err != nil || len(denialReasons) == 0 {
		t.Fatal("Unable to get the denial reasons for the refill request")
	}

	// now, lets go ahead and attempt to deny this refill request

	comment := "this is a comment"
	denyRefillRequest(refillRequest, doctor.AccountID.Int64(), comment, testData, t)

	refillRequest, err = testData.DataAPI.GetRefillRequestFromID(refillRequest.ID)
	if err != nil {
		t.Fatal("Unable to get refill request after approving request: " + err.Error())
	}

	if len(refillRequest.RxHistory) != 2 {
		t.Fatalf("Expected two items in the rx history of the refill request instead got %d", len(refillRequest.RxHistory))
	}

	if refillRequest.RxHistory[0].Status != api.RXRefillStatusDenied {
		t.Fatalf("Expected the refill request status to be %s but was %s instead: %+v", api.RXRefillStatusDenied, refillRequest.RxHistory[0].Status, refillRequest.RxHistory)
	}

	if refillRequest.Comments != comment {
		t.Fatalf("Expected the comment on the refill request to be '%s' but was '%s' instead", comment, refillRequest.Comments)
	}

	if refillRequest.PrescriptionID != deniedRefillRequestPrescriptionID {
		t.Fatalf("Expected the prescription id returned to be %d but instead it was %d", deniedRefillRequestPrescriptionID, refillRequest.PrescriptionID)
	}

	if refillRequest.DenialReason != denialReasons[0].DenialReason {
		t.Fatalf("Denial reason expected to be '%s' but is '%s' instead", denialReasons[0].DenialReason, refillRequest.DenialReason)
	}

	// doctor queue should be empty and the denied request should be in the completed tab
	completedItems, err := testData.DataAPI.GetCompletedItemsInDoctorQueue(doctor.ID.Int64())
	if err != nil {
		t.Fatal("Unable to get the completed items for the doctor: " + err.Error())
	}

	if len(completedItems) != 1 {
		t.Fatal("Expected there to be 1 completed item in the doctor's queue for the refill request that was just rejected")
	}

	if completedItems[0].EventType != api.DQEventTypeRefillRequest || completedItems[0].ItemID != refillRequest.ID ||
		completedItems[0].Status != api.DQItemStatusRefillDenied {
		t.Fatal("Completed item in the doctor's queue not in the expected state")
	}

	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	if err != nil {
		t.Fatal("Unable to get the pending items for the doctor: " + err.Error())
		return
	}

	if len(pendingItems) != 0 {
		t.Fatalf("Expected there to be no pending items in the doctor's queue instead there were %d", len(pendingItems))
	}

	// attempt to consume the message put into the queue
	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	for {
		consumed, err := statusWorker.Do()
		test.OK(t, err)
		if !consumed {
			break
		}
	}

	// now, the status of the refill request should be Sent
	refillStatusEvents, err := testData.DataAPI.GetRefillStatusEventsForRefillRequest(refillRequest.ID)
	if err != nil {
		t.Fatal("Unable to get refill status events for refill request: " + err.Error())
	}

	if len(refillStatusEvents) != 3 {
		t.Fatalf("Expected 3 refill status events for refill request but got %d", len(refillStatusEvents))
	}

	if refillStatusEvents[0].Status != api.RXRefillStatusDeleted {
		t.Fatal("Expected the top level item for the refill request to indicate that it was successfully sent to the pharmacy")
	}
}

// TestRefill_Deny is an integration test to test the system
// for the denial of a refill request.
func TestRefill_Deny(t *testing.T) {
	testRefill_Deny(false, t)
}

// TestRefill_Deny is an integration test to ensure that we allow
// denial of refill requests pertaining to controlled substances.
func TestRefill_Deny_ControlledSubstance(t *testing.T) {
	testRefill_Deny(true, t)
}

func TestRefill_Deny_DNTF_NoTreatment(t *testing.T) {

	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
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
		DOB:          encoding.Date{Month: 1, Day: 1, Year: 2000},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(12345),
	}

	err := testData.DataAPI.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIDForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemID := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      refillRequestQueueItemID,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
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
			DaysSupply:           encoding.NullInt64{},
			OTC:                  false,
			SubstitutionsAllowed: true,
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription),
				ErxPharmacyID:       1234,
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
				Int64Value: 5,
			}, PatientInstructions: "Take once daily",
			OTC: false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       1234,
				DoseSpotClinicianID: clinicianID,
			},
		},
	}

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}

	// Call the Consume method
	refillRxWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	refillRxWorker.Do()

	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	denialReasons, err := testData.DataAPI.GetRefillRequestDenialReasons()
	if err != nil || len(denialReasons) == 0 {
		t.Fatal("Unable to get the denial reasons for the refill request")
	}

	var dntfReason *api.RefillRequestDenialReason
	for _, denialReason := range denialReasons {
		if denialReason.DenialCode == api.RXRefillDNTFReasonCode {
			dntfReason = denialReason
			break
		}
	}

	if dntfReason == nil {
		t.Fatal("Unable to find DNTF reason in database: " + err.Error())
	}
	// now, lets go ahead and attempt to deny this refill request
	comment := "this is a test"
	requestData := doctorpkg.DoctorRefillRequestRequestData{
		RefillRequestID: refillRequest.ID,
		Action:          "deny",
		DenialReasonID:  dntfReason.ID,
		Comments:        comment,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		t.Fatal("Unable to marshal json into object: " + err.Error())
	}

	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.DoctorRefillRxURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected %d due to missing treatment object instead got %d ", http.StatusBadRequest, resp.StatusCode)
	}

	errorResponse := apiservice.ErrorResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		t.Fatal("Unable to unmarshal response body into json object: " + err.Error())
	}

	if errorResponse.DeveloperCode != apiservice.DeveloperErrorTreatmentMissingDNTF {
		t.Fatalf("Expected developer code of %d instead got %d", apiservice.DeveloperErrorTreatmentMissingDNTF, errorResponse.DeveloperCode)
	}

}

func setupRefill_Deny_DNTF(t *testing.T, testData *TestData, endErxStatus common.StatusEvent, toAddTemplatedTreatment bool) *common.Treatment {
	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
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
		DOB:          encoding.Date{Year: 1987, Month: 8, Day: 1},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(12345),
	}

	err := testData.DataAPI.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	comment := "this is a test"
	treatmentToAdd := common.Treatment{
		DrugInternalName: "Testing (If - This Works)",
		DrugDBIDs: map[string]string{
			erx.LexiSynonymTypeID: "12345",
			erx.LexiDrugSynID:     "123151",
			erx.LexiGenProductID:  "124151",
			erx.NDC:               "1415",
		},
		DosageStrength: "10 mg",
		DispenseValue:  1,
		DaysSupply:     encoding.NullInt64{},
		DispenseUnitID: encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		OTC:                 false,
		PatientInstructions: "patient instructions",
	}

	if toAddTemplatedTreatment {
		doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
		pDoctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
		test.OK(t, err)

		_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, pDoctor)

		treatmentTemplate := &common.DoctorTreatmentTemplate{
			Name:      "Favorite Treatment #1",
			Treatment: &treatmentToAdd,
		}

		treatmentTemplatesRequest := &doctor_treatment_plan.DoctorTreatmentTemplatesRequest{
			TreatmentTemplates: []*common.DoctorTreatmentTemplate{treatmentTemplate},
			TreatmentPlanID:    treatmentPlan.ID,
		}
		data, err := json.Marshal(&treatmentTemplatesRequest)
		test.OK(t, err)

		resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.DoctorTreatmentTemplatesURLPath, "application/json", bytes.NewBuffer(data), pDoctor.AccountID.Int64())
		test.OK(t, err)
		defer resp.Body.Close()
		test.Equals(t, http.StatusOK, resp.StatusCode)

		treatmentTemplatesResponse := &doctor_treatment_plan.DoctorTreatmentTemplatesResponse{}
		test.OK(t, json.NewDecoder(resp.Body).Decode(treatmentTemplatesResponse))

		treatmentToAdd.DoctorTreatmentTemplateID = treatmentTemplatesResponse.TreatmentTemplates[0].ID
	}

	testTime := time.Now()

	prescriptionIDForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemID := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      refillRequestQueueItemID,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              12345,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
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
			DaysSupply:           encoding.NullInt64{},
			SubstitutionsAllowed: true,
			ERx: &common.ERxData{
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription),
				ErxPharmacyID:       1234,
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
				Int64Value: 5,
			}, PatientInstructions: "Take once daily",
			OTC: false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       1234,
				DoseSpotClinicianID: clinicianID,
			},
		},
	}

	prescriptionIDForTreatment := int64(1234151515)
	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.PrescriptionIDsToReturn = []int64{prescriptionIDForTreatment}
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		prescriptionIDForTreatment: []common.StatusEvent{endErxStatus},
	}
	stubErxAPI.PharmacyToSendPrescriptionTo = pharmacyToReturn.SourceID

	refillRxWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	test.OK(t, refillRxWorker.Do())

	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	test.OK(t, err)

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	test.OK(t, err)

	denialReasons, err := testData.DataAPI.GetRefillRequestDenialReasons()
	if err != nil || len(denialReasons) == 0 {
		t.Fatal("Unable to get the denial reasons for the refill request")
	}

	var dntfReason *api.RefillRequestDenialReason
	for _, denialReason := range denialReasons {
		if denialReason.DenialCode == api.RXRefillDNTFReasonCode {
			dntfReason = denialReason
			break
		}
	}
	if dntfReason == nil {
		t.Fatal("Unable to find DNTF reason in database: " + err.Error())
	}

	// now, lets go ahead and attempt to deny this refill request

	requestData := doctorpkg.DoctorRefillRequestRequestData{
		RefillRequestID: refillRequest.ID,
		Action:          "deny",
		DenialReasonID:  dntfReason.ID,
		Comments:        comment,
		Treatment:       &treatmentToAdd,
	}

	jsonData, err := json.Marshal(requestData)
	test.OK(t, err)
	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.DoctorRefillRxURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountID.Int64())
	test.OK(t, err)
	resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// get refill request to ensure that it was denied
	refillRequest, err = testData.DataAPI.GetRefillRequestFromID(refillRequest.ID)
	test.OK(t, err)
	test.Equals(t, 2, len(refillRequest.RxHistory))
	test.Equals(t, api.RXRefillStatusDenied, refillRequest.RxHistory[0].Status)

	// get unlinked treatment
	unlinkedDNTFTreatmentStatusEvents, err := testData.DataAPI.GetErxStatusEventsForDNTFTreatmentBasedOnPatientID(refillRequest.Patient.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(unlinkedDNTFTreatmentStatusEvents))

	unlinkedTreatment, err := testData.DataAPI.GetUnlinkedDNTFTreatment(unlinkedDNTFTreatmentStatusEvents[0].ItemID)
	test.OK(t, err)
	test.Equals(t, prescriptionIDForTreatment, unlinkedTreatment.ERx.PrescriptionID.Int64())
	test.Equals(t, refillRequest.RequestedPrescription.ERx.Pharmacy.LocalID, unlinkedTreatment.ERx.Pharmacy.LocalID)
	test.Equals(t, 2, len(unlinkedTreatment.ERx.RxHistory))
	for _, ev := range unlinkedTreatment.ERx.RxHistory {
		if ev.InternalStatus == api.StatusInactive && ev.Status != api.ERXStatusNewRXFromDNTF {
			t.Fatalf("Expected top level item in rx history to be %s instead it was %s", api.ERXStatusNewRXFromDNTF, ev.Status)
		}
	}

	// check dntf mapping to ensure that there is an entry
	var dntfMappingCount int64
	test.OK(t, testData.DB.QueryRow(`SELECT count(*) FROM dntf_mapping`).Scan(&dntfMappingCount))
	test.Equals(t, int64(1), dntfMappingCount)

	// create an artificial delay in between the newRXDNTF and Sending state such that there is a guarantee that NewRXDNTF happens after Sending.
	res, err := testData.DB.Exec(`UPDATE unlinked_dntf_treatment_status_events SET creation_date = ? WHERE unlinked_dntf_treatment_id = ? and erx_status = ?`,
		time.Now().Add(-1*time.Minute), unlinkedTreatment.ID.Int64(), api.ERXStatusSending)
	test.OK(t, err)
	n, err := res.RowsAffected()
	test.OK(t, err)
	test.Equals(t, int64(1), n)

	res, err = testData.DB.Exec(`UPDATE unlinked_dntf_treatment_status_events SET creation_date = ? WHERE unlinked_dntf_treatment_id = ? and erx_status = ?`,
		time.Now().Add(-5*time.Minute), unlinkedTreatment.ID.Int64(), api.ERXStatusNewRXFromDNTF)
	test.OK(t, err)
	n, err = res.RowsAffected()
	test.OK(t, err)
	test.Equals(t, int64(1), n)

	// check erx status to be sent once its sent
	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	for {
		consumed, err := statusWorker.Do()
		test.OK(t, err)
		if !consumed {
			break
		}
	}

	unlinkedTreatment, err = testData.DataAPI.GetUnlinkedDNTFTreatment(unlinkedTreatment.ID.Int64())
	test.OK(t, err)
	return unlinkedTreatment
}

// TestRefill_Deny_DNTF_NonSprucePatient is an integration test
// to ensure that the Deny New Request To Follow experience for a refill
// request works as expected for a non-Spruce patient.
func TestRefill_Deny_DNTF_NonSprucePatient(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	unlinkedTreatment := setupRefill_Deny_DNTF(t, testData, common.StatusEvent{Status: api.ERXStatusSent}, false)
	if len(unlinkedTreatment.ERx.RxHistory) != 3 {
		for _, e := range unlinkedTreatment.ERx.RxHistory {
			t.Logf("%+v", e)
		}
		t.Fatalf("Expected 3 events from rx history of unlinked treatment instead got %d", len(unlinkedTreatment.ERx.RxHistory))
	}

	for _, unlinkedTreatmentStatusEvent := range unlinkedTreatment.ERx.RxHistory {
		if unlinkedTreatmentStatusEvent.InternalStatus == api.StatusActive && unlinkedTreatmentStatusEvent.Status != api.ERXStatusSent {
			t.Fatalf("Expected status %s for top level status of unlinked treatment but got %s", api.ERXStatusSent, unlinkedTreatmentStatusEvent.Status)
		}
	}
}

// TestRefill_Deny_DNTF_NonSprucePatient_FromTemplate is an integration test
// to ensure that the Deny New Request To Follow experience for a refill
// request works well for a non-Spruce patient when the doctor attempts to
// respond with a new request taht is picked from a templated treatment.
func TestRefill_Deny_DNTF_NonSprucePatient_FromTemplate(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	unlinkedTreatment := setupRefill_Deny_DNTF(t, testData, common.StatusEvent{Status: api.ERXStatusSent}, true)
	if len(unlinkedTreatment.ERx.RxHistory) != 3 {
		for _, e := range unlinkedTreatment.ERx.RxHistory {
			t.Logf("%+v", e)
		}
		t.Fatalf("Expected 3 events from rx history of unlinked treatment instead got %d", len(unlinkedTreatment.ERx.RxHistory))
	}

	for _, unlinkedTreatmentStatusEvent := range unlinkedTreatment.ERx.RxHistory {
		if unlinkedTreatmentStatusEvent.InternalStatus == api.StatusActive && unlinkedTreatmentStatusEvent.Status != api.ERXStatusSent {
			t.Fatalf("Expected status %s for top level status of unlinked treatment but got %s", api.ERXStatusSent, unlinkedTreatmentStatusEvent.Status)
		}
	}
}

// TestRefill_Deny_DNTF_NonSprucePatient_Error is an integration test
// to ensure that Deny New Request To Follow experience for a refill
// request works well for a non-Spruce patient when the new prescription
// routed in response to the refill request has an error in being routed
// to the pharmacy.
func TestRefill_Deny_DNTF_NonSprucePatient_Error(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	errorMessage := "this is a test error message"
	unlinkedTreatment := setupRefill_Deny_DNTF(t, testData, common.StatusEvent{Status: api.ERXStatusError, StatusDetails: errorMessage}, false)

	if len(unlinkedTreatment.ERx.RxHistory) != 3 {
		for _, e := range unlinkedTreatment.ERx.RxHistory {
			t.Logf("%+v", e)
		}
		t.Fatalf("Expected 3 events from rx history of unlinked treatment instead got %d", len(unlinkedTreatment.ERx.RxHistory))
	}

	for _, unlinkedTreatmentStatusEvent := range unlinkedTreatment.ERx.RxHistory {
		if unlinkedTreatmentStatusEvent.InternalStatus == api.StatusActive {
			if unlinkedTreatmentStatusEvent.Status != api.ERXStatusError {
				t.Fatalf("Expected status %s for top level status of unlinked treatment but got %s", api.ERXStatusSent, unlinkedTreatmentStatusEvent.Status)
			}
			if unlinkedTreatmentStatusEvent.StatusDetails != errorMessage {
				t.Fatalf("Expected the error message for the status to be '%s' but it was '%s' instead", errorMessage, unlinkedTreatmentStatusEvent.StatusDetails)
			}
		}
	}

	// check if this results in an item in the doctor queue
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(unlinkedTreatment.Doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
	if pendingItems[0].EventType != api.DQEventTypeUnlinkedDNTFTransmissionError {
		t.Fatalf("Expected event type of item in doctor queue to be %s but was %s instead", api.DQEventTypeUnlinkedDNTFTransmissionError, pendingItems[0].EventType)
	}

	params := &url.Values{}
	params.Set("unlinked_dntf_treatment_id", strconv.FormatInt(unlinkedTreatment.ID.Int64(), 10))

	resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.DoctorRXErrorResolveURLPath, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()), unlinkedTreatment.Doctor.AccountID.Int64())
	test.OK(t, err)
	resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	pendingItems, err = testData.DataAPI.GetPendingItemsInDoctorQueue(unlinkedTreatment.Doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 0, len(pendingItems))

	completedItems, err := testData.DataAPI.GetCompletedItemsInDoctorQueue(unlinkedTreatment.Doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(completedItems))
}

func setupRefill_Deny_DNTF_ExistingPatient(t *testing.T, testData *TestData, endErxStatus common.StatusEvent, toAddTemplatedTreatment bool) *common.Treatment {
	// create doctor with clinicianId specified
	doctor := createDoctorWithClinicianID(testData, t)
	doctorClient := DoctorClient(testData, t, doctor.ID.Int64())

	pv, _ := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	erxPatientID := int64(60)

	// add an erx patient id to the patient
	err = testData.DataAPI.UpdatePatient(patient.ID.Int64(), &api.PatientUpdate{
		ERxID: &erxPatientID,
	}, false)
	if err != nil {
		t.Fatal("Unable to update patient with erx patient id : " + err.Error())
	}

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err = testData.DataAPI.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	// start a new treatemtn plan for the patient visit
	treatmentPlan, err := doctorClient.PickTreatmentPlanForVisit(pv.PatientVisitID, nil)
	test.OK(t, err)
	treatmentPlanID := treatmentPlan.ID.Int64()

	comment := "this is a test"
	treatmentToAdd := common.Treatment{
		DrugInternalName: "Testing (If - This Works)",
		DrugDBIDs: map[string]string{
			erx.LexiSynonymTypeID: "12345",
			erx.LexiDrugSynID:     "123151",
			erx.LexiGenProductID:  "124151",
			erx.NDC:               "1415",
		},
		DosageStrength: "10 mg",
		DispenseValue:  1,
		DispenseUnitID: encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		DaysSupply:          encoding.NullInt64{},
		OTC:                 false,
		PatientInstructions: "patient instructions",
	}

	if toAddTemplatedTreatment {
		treatmentTemplate := &common.DoctorTreatmentTemplate{
			Name:      "Favorite Treatment #1",
			Treatment: &treatmentToAdd,
		}

		treatmentTemplatesRequest := &doctor_treatment_plan.DoctorTreatmentTemplatesRequest{
			TreatmentTemplates: []*common.DoctorTreatmentTemplate{treatmentTemplate},
			TreatmentPlanID:    encoding.NewObjectID(treatmentPlanID),
		}
		data, err := json.Marshal(&treatmentTemplatesRequest)
		test.OK(t, err)
		resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.DoctorTreatmentTemplatesURLPath, "application/json", bytes.NewBuffer(data), doctor.AccountID.Int64())
		test.OK(t, err)
		defer resp.Body.Close()
		test.Equals(t, http.StatusOK, resp.StatusCode)

		treatmentTemplatesResponse := &doctor_treatment_plan.DoctorTreatmentTemplatesResponse{}
		test.OK(t, json.NewDecoder(resp.Body).Decode(treatmentTemplatesResponse))

		treatmentToAdd.DoctorTreatmentTemplateID = treatmentTemplatesResponse.TreatmentTemplates[0].ID
	}

	testTime := time.Now()

	treatment1 := &common.Treatment{
		DrugDBIDs: map[string]string{
			erx.LexiDrugSynID:     "1234",
			erx.LexiGenProductID:  "12345",
			erx.LexiSynonymTypeID: "123556",
			erx.NDC:               "2415",
		},
		DrugName:                "Testing",
		DrugRoute:               "topical",
		DrugForm:                "gel",
		DosageStrength:          "10 mg",
		DispenseValue:           5,
		DispenseUnitDescription: "Tablet",
		DispenseUnitID:          encoding.NewObjectID(19),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		SubstitutionsAllowed: false,
		DaysSupply:           encoding.NullInt64{},
		PatientInstructions:  "Take once daily",
		OTC:                  false,
		ERx: &common.ERxData{
			PrescriptionID:     encoding.NewObjectID(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyID:      1234,
			PharmacyLocalID:    encoding.NewObjectID(pharmacyToReturn.LocalID),
			ErxLastDateFilled:  &testTime,
		},
	}

	// add this treatment to the treatment plan
	err = testData.DataAPI.AddTreatmentsForTreatmentPlan([]*common.Treatment{treatment1}, doctor.ID.Int64(), treatmentPlanID, patient.ID.Int64())
	test.OK(t, err)

	// insert erxStatusEvent for this treatment to indicate that it was sent
	_, err = testData.DB.Exec(`INSERT INTO erx_status_events (treatment_id, erx_status, creation_date, status) VALUES (?,?,?,?)`,
		treatment1.ID.Int64(), api.ERXStatusSent, testTime, "ACTIVE")
	test.OK(t, err)

	// update the treatment with prescription id and pharmacy id for where prescription was routed
	res, err := testData.DB.Exec(`UPDATE treatment SET erx_id = ?, pharmacy_id = ? where id = ?`,
		treatment1.ERx.PrescriptionID.Int64(), pharmacyToReturn.LocalID, treatment1.ID.Int64())
	test.OK(t, err)
	n, err := res.RowsAffected()
	test.Equals(t, int64(1), n)

	prescriptionIDForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemID := int64(12345)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      refillRequestQueueItemID,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              erxPatientID,
		PatientAddedForRequest:    false,
		RequestDateStamp:          testTime,
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
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription),
				ErxPharmacyID:       1234,
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
				Int64Value: 5,
			},
			PatientInstructions: "Take once daily",
			OTC:                 false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       1234,
				DoseSpotClinicianID: clinicianID,
			},
		},
	}

	prescriptionIDForTreatment := int64(15515616)
	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}
	stubErxAPI.PrescriptionIDsToReturn = []int64{prescriptionIDForTreatment}
	stubErxAPI.PharmacyToSendPrescriptionTo = pharmacyToReturn.SourceID
	stubErxAPI.ExpectedRxReferenceNumber = strconv.FormatInt(refillRequestItem.RxRequestQueueItemID, 10)
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		prescriptionIDForTreatment: []common.StatusEvent{endErxStatus},
	}

	// Call the Consume method
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	test.OK(t, refillRXWorker.Do())

	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error())
	}

	denialReasons, err := testData.DataAPI.GetRefillRequestDenialReasons()
	if err != nil || len(denialReasons) == 0 {
		t.Fatal("Unable to get the denial reasons for the refill request")
	}

	var dntfReason *api.RefillRequestDenialReason
	for _, denialReason := range denialReasons {
		if denialReason.DenialCode == api.RXRefillDNTFReasonCode {
			dntfReason = denialReason
			break
		}
	}

	if dntfReason == nil {
		t.Fatal("Unable to find DNTF reason in database: " + err.Error())
	}

	// now, lets go ahead and attempt to deny this refill request

	requestData := doctorpkg.DoctorRefillRequestRequestData{
		RefillRequestID: refillRequest.ID,
		Action:          "deny",
		DenialReasonID:  dntfReason.ID,
		Comments:        comment,
		Treatment:       &treatmentToAdd,
	}

	jsonData, err := json.Marshal(requestData)
	test.OK(t, err)
	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.DoctorRefillRxURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountID.Int64())
	test.OK(t, err)
	resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// get refill request to ensure that it was denied
	refillRequest, err = testData.DataAPI.GetRefillRequestFromID(refillRequest.ID)
	if err != nil {
		t.Fatalf("Unable to get refill request from id: %+v", err)
	}

	if len(refillRequest.RxHistory) != 2 {
		t.Fatalf("Expected there to be 2 refill request events instead there were %d", len(refillRequest.RxHistory))
	}

	if refillRequest.RxHistory[0].Status != api.RXRefillStatusDenied {
		t.Fatalf("Expected top level refill request status of %s instead got %s", refillRequestItem.RxHistory[0].Status, api.RXRefillStatusDenied)
	}

	// get unlinked treatment
	unlinkedDNTFTreatmentStatusEvents, err := testData.DataAPI.GetErxStatusEventsForDNTFTreatmentBasedOnPatientID(refillRequest.Patient.ID.Int64())
	if err != nil {
		t.Fatalf("Unable to get status events for dntf treatment: %+v", err)
	}
	if len(unlinkedDNTFTreatmentStatusEvents) != 0 {
		t.Fatalf("Expected 0 status events for unlinked dntf treatments instead got %d", len(unlinkedDNTFTreatmentStatusEvents))
	}

	// check dntf mapping to ensure that there is an entry
	var dntfMappingCount int64
	test.OK(t, testData.DB.QueryRow(`select count(*) from dntf_mapping`).Scan(&dntfMappingCount))
	test.Equals(t, int64(1), dntfMappingCount)

	treatments, err := testData.DataAPI.GetTreatmentsBasedOnTreatmentPlanID(treatmentPlanID)
	test.OK(t, err)
	test.Equals(t, 2, len(treatments))

	var linkedTreatment *common.Treatment
	for _, treatment := range treatments {
		if treatment.ERx.PrescriptionID.Int64() == prescriptionIDForTreatment {
			linkedTreatment = treatment
			break
		}
	}

	if linkedTreatment == nil {
		t.Fatalf("Unable to find the treatment that was added as a result of DNTF")
	}

	if toAddTemplatedTreatment {
		if !linkedTreatment.DoctorTreatmentTemplateID.IsValid || linkedTreatment.DoctorTreatmentTemplateID.Int64() == 0 {
			t.Fatal("Expected there to exist a doctor template id given that the treatment was created from a template but there wasnt one")
		}
	}

	// the treatment as a result of DNTF, if linked, should map back to the original treatemtn plan
	// associated with the originating treatment for the refill request
	if !linkedTreatment.TreatmentPlanID.IsValid || linkedTreatment.TreatmentPlanID.Int64() != treatmentPlanID {
		t.Fatalf("Expected the linked treatment to map back to the original treatment but it didnt")
	}

	if len(linkedTreatment.ERx.RxHistory) != 2 {
		t.Fatalf("Expected there to be 2 events for this linked dntf treatment, instead got %d", len(linkedTreatment.ERx.RxHistory))
	}

	for _, linkedTreatmentStatus := range linkedTreatment.ERx.RxHistory {
		if linkedTreatmentStatus.InternalStatus == api.StatusInactive && linkedTreatmentStatus.Status != api.ERXStatusNewRXFromDNTF {
			t.Fatalf("Expected the first event for the linked treatment to be %s instead it was %s", api.ERXStatusNewRXFromDNTF, linkedTreatmentStatus.Status)
		}
	}

	// create an artificial delay in between the newRXDNTF and Sending state such that there is a guarantee that NewRXDNTF happens after Sending.
	_, err = testData.DB.Exec(`UPDATE erx_status_events SET creation_date = ? WHERE treatment_id = ? AND erx_status = ?`,
		time.Now().Add(-1*time.Minute), linkedTreatment.ID.Int64(), api.ERXStatusSending)
	test.OK(t, err)

	_, err = testData.DB.Exec(`UPDATE erx_status_events SET creation_date = ? WHERE treatment_id = ? AND erx_status = ?`,
		time.Now().Add(-5*time.Minute), linkedTreatment.ID.Int64(), api.ERXStatusNewRXFromDNTF)
	test.OK(t, err)

	// check erx status to be sent once its sent
	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	for {
		consumed, err := statusWorker.Do()
		test.OK(t, err)
		if !consumed {
			break
		}
	}

	linkedTreatment, err = testData.DataAPI.GetTreatmentBasedOnPrescriptionID(prescriptionIDForTreatment)
	test.OK(t, err)
	return linkedTreatment
}

// TestRefill_Deny_DNTF_ExistingPatient is an integration test to ensure
// that the DNTF process works well for an existing patient with a refill request
// coming in for an existing treatment.
func TestRefill_Deny_DNTF_ExistingPatient(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	linkedTreatment := setupRefill_Deny_DNTF_ExistingPatient(t, testData, common.StatusEvent{Status: api.ERXStatusSent}, false)

	if len(linkedTreatment.ERx.RxHistory) != 3 {
		t.Fatalf("Expected 3 events for linked treatment instead got %d", len(linkedTreatment.ERx.RxHistory))
	}

	for _, linkedTreatmentStatusEvent := range linkedTreatment.ERx.RxHistory {
		if linkedTreatmentStatusEvent.InternalStatus == api.StatusActive && linkedTreatmentStatusEvent.Status != api.ERXStatusSent {
			t.Fatalf("Expected the latest event for the linked treatment to be %s instead it was %s", api.ERXStatusSent, linkedTreatmentStatusEvent.Status)
		}
	}
}

// TestRefill_Deny_DNTF_ExistingPatient_TemplatedTreatment is an integration test to ensure
// that the DNTF process works well for an existing patient with a refill request
// coming in for an existing treatment where the new rx is picked from a templated treatment.
func TestRefill_Deny_DNTF_ExistingPatient_TemplatedTreatment(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	linkedTreatment := setupRefill_Deny_DNTF_ExistingPatient(t, testData, common.StatusEvent{Status: api.ERXStatusSent}, true)

	if len(linkedTreatment.ERx.RxHistory) != 3 {
		t.Fatalf("Expected 3 events for linked treatment instead got %v", linkedTreatment.ERx.RxHistory)
	}

	for _, linkedTreatmentStatusEvent := range linkedTreatment.ERx.RxHistory {
		if linkedTreatmentStatusEvent.InternalStatus == api.StatusActive && linkedTreatmentStatusEvent.Status != api.ERXStatusSent {
			t.Fatalf("Expected the latest event for the linked treatment to be %s instead it was %s", api.ERXStatusSent, linkedTreatmentStatusEvent.Status)
		}
	}
}

// TestRefill_DNTF_ExistingPatient_Error is an integration test to ensure
// that the DNTF process works well for an existing patient where the new rx created
// has an error in being routed to the pharmacy.
func TestRefill_DNTF_ExistingPatient_Error(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	errorMessage := "this is a test error message"
	linkedTreatment := setupRefill_Deny_DNTF_ExistingPatient(t, testData, common.StatusEvent{Status: api.ERXStatusError, StatusDetails: errorMessage}, false)

	if len(linkedTreatment.ERx.RxHistory) != 3 {
		t.Fatalf("Expected 3 events for linked treatment instead got %d", len(linkedTreatment.ERx.RxHistory))
	}

	for _, linkedTreatmentStatusEvent := range linkedTreatment.ERx.RxHistory {
		if linkedTreatmentStatusEvent.InternalStatus == api.StatusActive {
			if linkedTreatmentStatusEvent.Status != api.ERXStatusError {
				t.Fatalf("Expected the latest event for the linked treatment to be %s instead it was %s", api.ERXStatusError, linkedTreatmentStatusEvent.Status)
			}

			if linkedTreatmentStatusEvent.StatusDetails != errorMessage {
				t.Fatalf("Expected the status message to be '%s instead it was '%s'", errorMessage, linkedTreatmentStatusEvent.StatusDetails)
			}
		}
	}

	// there should be one item in the doctor's queue relating to a transmission error
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(linkedTreatment.Doctor.ID.Int64())
	test.OK(t, err)

	if len(pendingItems) != 2 {
		t.Fatalf("Expected there to be 2 pending items but got %d", len(pendingItems))
	} else if pendingItems[1].EventType != api.DQEventTypeTransmissionError {
		t.Fatalf("Expected the one item in the doctors queue to be of type %s instead it was of type %s", api.DQEventTypeTransmissionError, pendingItems[1].EventType)
	}
}

// TestRefill_Status_MultipleRefills is an integration test to test the status of multiple refill
// requests at once to ensure that the logic for the refill request worker works as expected
// when working with refill requests in a batch.
func TestRefill_Status_MultipleRefills(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)

	approvedRefillRequestPrescriptionID := int64(101010)
	approvedRefillAmount := int64(10)

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
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
		DOB:          encoding.Date{Month: 1, Year: 1967, Day: 1},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(12345),
	}

	err := testData.DataAPI.AddPharmacy(pharmacyToReturn)
	if err != nil {
		t.Fatal("Unable to store pharmacy in db: " + err.Error())
	}

	testTime := time.Now()

	prescriptionIDForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	refillRequestQueueItemID := int64(12345)
	refillRequests := make([]*common.RefillRequestItem, 0)
	for i := int64(0); i < 4; i++ {
		// Get StubErx to return refill requests in the refillRequest call
		refillRequestItem := &common.RefillRequestItem{
			RxRequestQueueItemID:      refillRequestQueueItemID + i,
			ReferenceNumber:           "TestReferenceNumber",
			PharmacyRxReferenceNumber: "TestRxReferenceNumber",
			ErxPatientID:              12345,
			PatientAddedForRequest:    true,
			RequestDateStamp:          testTime,
			ClinicianID:               clinicianID,
			RequestedPrescription: &common.Treatment{
				DrugDBIDs: map[string]string{
					erx.LexiDrugSynID:     "1234",
					erx.LexiGenProductID:  "12345",
					erx.LexiSynonymTypeID: "123556",
					erx.NDC:               "2415",
				},
				DosageStrength:       "10 mg",
				DaysSupply:           encoding.NullInt64{},
				DispenseValue:        5,
				OTC:                  false,
				SubstitutionsAllowed: true,
				ERx: &common.ERxData{
					ErxSentDate:         &fiveMinutesBeforeTestTime,
					DoseSpotClinicianID: clinicianID,
					PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription + i),
					ErxPharmacyID:       1234,
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
					Int64Value: 5,
				}, PatientInstructions: "Take once daily",
				OTC: false,
				ERx: &common.ERxData{
					DoseSpotClinicianID: clinicianID,
					PrescriptionID:      encoding.NewObjectID(5504),
					PrescriptionStatus:  "Requested",
					ErxPharmacyID:       1234,
					ErxLastDateFilled:   &testTime,
				},
			},
		}
		refillRequests = append(refillRequests, refillRequestItem)
	}

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequests[0]}
	stubErxAPI.RefillRequestPrescriptionIDs = map[int64]int64{
		refillRequestQueueItemID: approvedRefillRequestPrescriptionID,
	}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		approvedRefillRequestPrescriptionID: []common.StatusEvent{{Status: api.ERXStatusSent}},
	}

	// Call the Consume method so that the first refill request gets added to the system
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	refillRXWorker.Do()

	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	test.OK(t, err)

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	test.OK(t, err)

	// lets go ahead and approve this refill request
	comment := "this is a test"
	requestData := doctorpkg.DoctorRefillRequestRequestData{
		RefillRequestID:      refillRequest.ID,
		Action:               "approve",
		ApprovedRefillAmount: approvedRefillAmount,
		Comments:             comment,
	}

	jsonData, err := json.Marshal(&requestData)
	test.OK(t, err)
	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.DoctorRefillRxURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountID.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	if _, err := testData.DataAPI.GetRefillRequestFromID(refillRequest.ID); err != nil {
		t.Fatal("Unable to get refill request after approving request: " + err.Error())
	}

	// now lets go ahead and ensure that the refill request is successfully sent to the pharmacy
	statusWorker := app_worker.NewERxStatusWorker(
		testData.DataAPI,
		stubErxAPI,
		testData.Config.Dispatcher,
		testData.Config.ERxStatusQueue,
		testData.Config.MetricsRegistry)
	for {
		consumed, err := statusWorker.Do()
		test.OK(t, err)
		if !consumed {
			break
		}
	}

	// now, lets go ahead and get 3 refill requests queued up for the clinic
	stubErxAPI.RefillRxRequestQueueToReturn = refillRequests[1:]
	stubErxAPI.RefillRequestPrescriptionIDs = map[int64]int64{
		refillRequestQueueItemID:     approvedRefillRequestPrescriptionID,
		refillRequestQueueItemID + 1: approvedRefillRequestPrescriptionID + 1,
		refillRequestQueueItemID + 2: approvedRefillRequestPrescriptionID + 2,
		refillRequestQueueItemID + 3: approvedRefillRequestPrescriptionID + 3,
	}
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		approvedRefillRequestPrescriptionID:     []common.StatusEvent{common.StatusEvent{Status: api.ERXStatusSent}},
		approvedRefillRequestPrescriptionID + 1: []common.StatusEvent{common.StatusEvent{Status: api.ERXStatusSent}},
		approvedRefillRequestPrescriptionID + 2: []common.StatusEvent{common.StatusEvent{Status: api.ERXStatusSent}},
		approvedRefillRequestPrescriptionID + 3: []common.StatusEvent{common.StatusEvent{Status: api.ERXStatusSent}},
	}

	test.OK(t, refillRXWorker.Do())

	refillRequestStatuses, err = testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	test.OK(t, err)
	test.Equals(t, 3, len(refillRequestStatuses))

	// go ahead and approve all remaining refill requests
	for i := 0; i < len(refillRequestStatuses); i++ {
		requestData.RefillRequestID = refillRequestStatuses[i].ItemID
		jsonData, err = json.Marshal(&requestData)
		test.OK(t, err)
		resp, err = testData.AuthPut(testData.APIServer.URL+apipaths.DoctorRefillRxURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountID.Int64())
		test.OK(t, err)
		resp.Body.Close()
		test.Equals(t, http.StatusOK, resp.StatusCode)
	}

	// now lets go ahead and ensure that the refill request is successfully sent to the pharmacy
	consumed, err := statusWorker.Do()
	test.OK(t, err)
	test.Equals(t, true, consumed)

	// all 3 refill requests should not have 3 items in the rx history
	refillRequestStatusEvents, err := testData.DataAPI.GetRefillStatusEventsForRefillRequest(refillRequestStatuses[0].ItemID)
	test.OK(t, err)
	test.Equals(t, 3, len(refillRequestStatusEvents))

	refillRequestStatusEvents, err = testData.DataAPI.GetRefillStatusEventsForRefillRequest(refillRequestStatuses[1].ItemID)
	test.OK(t, err)
	test.Equals(t, 3, len(refillRequestStatusEvents))

	refillRequestStatusEvents, err = testData.DataAPI.GetRefillStatusEventsForRefillRequest(refillRequestStatuses[2].ItemID)
	test.OK(t, err)
	test.Equals(t, 3, len(refillRequestStatusEvents))

	stubSqs := testData.Config.ERxStatusQueue.QueueService.(*awsutil.SQS)
	test.Equals(t, 2, len(stubSqs.Messages[testData.Config.ERxStatusQueue.QueueURL]))

	// now lets go ahead and ensure that the refill request is successfully sent to the pharmacy
	consumed, err = statusWorker.Do()
	test.OK(t, err)
	test.Equals(t, true, consumed)
	test.Equals(t, 1, len(stubSqs.Messages[testData.Config.ERxStatusQueue.QueueURL]))

	// now lets go ahead and ensure that the refill request is successfully sent to the pharmacy
	consumed, err = statusWorker.Do()
	test.OK(t, err)
	test.Equals(t, true, consumed)
	test.Equals(t, 0, len(stubSqs.Messages[testData.Config.ERxStatusQueue.QueueURL]))
}

// TestRefill_DifferentPharmacy tests the refill request flow through the system
// when the refill request comes from a different pharmacy than the pharmacy where the medication was originaly dispensed.
func TestRefill_DifferentPharmacy(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)
	doctorClient := DoctorClient(testData, t, doctor.ID.Int64())
	signedupPatientResponse := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	erxPatientID := int64(60)

	// add an erx patient id to the patient
	err := testData.DataAPI.UpdatePatient(signedupPatientResponse.Patient.ID.Int64(), &api.PatientUpdate{
		ERxID: &erxPatientID,
	}, false)
	if err != nil {
		t.Fatal("Unable to update patient with erx patient id : " + err.Error())
	}

	// add pharmacy to database so that it can be linked to treatment that is added
	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	anotherPharmacyToAdd := &pharmacy.PharmacyData{
		SourceID:     12345678,
		Source:       pharmacy.PharmacySourceSurescripts,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	test.OK(t, testData.DataAPI.AddPharmacy(pharmacyToReturn))
	test.OK(t, testData.DataAPI.AddPharmacy(anotherPharmacyToAdd))

	pv, _ := CreatePatientVisitAndPickTP(t, testData, signedupPatientResponse.Patient, doctor)

	// start a new treatment plan for the patient visit
	tp, err := doctorClient.PickTreatmentPlanForVisit(pv.PatientVisitID, nil)
	test.OK(t, err)
	treatmentPlanID := tp.ID.Int64()

	testTime := time.Now()

	treatment1 := &common.Treatment{
		DrugDBIDs: map[string]string{
			erx.LexiDrugSynID:     "1234",
			erx.LexiGenProductID:  "12345",
			erx.LexiSynonymTypeID: "123556",
			erx.NDC:               "2415",
		},
		DrugName:                "Testing",
		DrugRoute:               "topical",
		DrugForm:                "cream",
		DosageStrength:          "10 mg",
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
		}, PatientInstructions: "Take once daily",
		OTC: false,
		ERx: &common.ERxData{
			ErxLastDateFilled:  &testTime,
			PrescriptionID:     encoding.NewObjectID(5504),
			PrescriptionStatus: "Requested",
			ErxPharmacyID:      1234,
			PharmacyLocalID:    encoding.NewObjectID(pharmacyToReturn.LocalID),
		},
	}

	// add this treatment to the treatment plan
	err = testData.DataAPI.AddTreatmentsForTreatmentPlan([]*common.Treatment{treatment1}, doctor.ID.Int64(), treatmentPlanID, signedupPatientResponse.Patient.ID.Int64())
	test.OK(t, err)

	// insert erxStatusEvent for this treatment to indicate that it was sent
	_, err = testData.DB.Exec(`INSERT INTO erx_status_events (treatment_id, erx_status, creation_date, status) VALUES (?,?,?,?)`, treatment1.ID.Int64(), api.ERXStatusSent, testTime, "ACTIVE")
	test.OK(t, err)

	// update the treatment with prescription id and pharmacy id for where prescription was routed
	_, err = testData.DB.Exec(`UPDATE treatment SET erx_id = ?, pharmacy_id = ? WHERE id = ?`, treatment1.ERx.PrescriptionID.Int64(), pharmacyToReturn.LocalID, treatment1.ID.Int64())
	test.OK(t, err)

	prescriptionIDForRequestedPrescription := int64(123456)
	fiveMinutesBeforeTestTime := testTime.Add(-5 * time.Minute)
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      12345,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              erxPatientID,
		PatientAddedForRequest:    false,
		RequestDateStamp:          testTime,
		ClinicianID:               clinicianID,
		RequestedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				erx.LexiDrugSynID:     "1234",
				erx.LexiGenProductID:  "12345",
				erx.LexiSynonymTypeID: "123556",
				erx.NDC:               "2415",
			},
			DosageStrength:       "10 mg",
			DaysSupply:           encoding.NullInt64{},
			DispenseValue:        5,
			OTC:                  false,
			SubstitutionsAllowed: true,
			ERx: &common.ERxData{
				DoseSpotClinicianID: clinicianID,
				ErxSentDate:         &fiveMinutesBeforeTestTime,
				PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription),
				ErxPharmacyID:       1234,
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
				Int64Value: 5,
			}, PatientInstructions: "Take once daily",
			OTC: false,
			ERx: &common.ERxData{
				ErxLastDateFilled:   &testTime,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       12345678,
				DoseSpotClinicianID: clinicianID,
			},
		},
	}

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}

	// Call the Consume method
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	test.OK(t, refillRXWorker.Do())

	var count int64
	test.OK(t, testData.DB.QueryRow(`SELECT count(*) FROM requested_treatment`).Scan(&count))
	test.Equals(t, false, 0 == count)

	// There should be a status entry in the refill_request_status table
	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	test.OK(t, err)
	test.Equals(t, 1, len(refillRequestStatuses))

	if refillRequestStatuses[0].ItemID != refillRequestItem.ID ||
		refillRequestStatuses[0].Status != api.RXRefillStatusRequested {
		t.Fatal("Refill request status not in expected state")
	}

	// There should be a pending entry in the doctor's queue
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)

	if len(pendingItems) != 2 {
		t.Fatalf("Expected there to exist 2 pending items but got %d", len(pendingItems))
	} else if pendingItems[1].EventType != api.DQEventTypeRefillRequest ||
		pendingItems[1].ItemID != refillRequestStatuses[0].ItemID {
		t.Fatal("Pending item found in the doctor's queue is not the expected item")
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	test.OK(t, err)
	test.Equals(t, false, refillRequest.DispensedPrescription == nil)
	test.Equals(t, false, refillRequest.RequestedPrescription == nil)
	test.Equals(t, false, refillRequest.RequestedPrescription.OriginatingTreatmentID == 0)
	test.Equals(t, false, refillRequest.Patient == nil)
	test.Equals(t, api.PatientRegistered, refillRequest.Patient.Status)
	if refillRequest.RequestedPrescription.ERx.Pharmacy == nil || refillRequest.DispensedPrescription.ERx.Pharmacy == nil {
		t.Fatal("Expected pharmacy object to be present for requested and dispensed prescriptions")
	}
	if refillRequest.RequestedPrescription.ERx.Pharmacy.SourceID == refillRequest.DispensedPrescription.ERx.Pharmacy.SourceID {
		t.Fatal("Expected the pharmacies to be different between the requested and the dispensed prescriptions")
	}
}

func TestRefill_ExistingPatient_NonexistentTreatment(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)

	signedupPatientResponse := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	erxPatientID := int64(60)

	// add an erx patient id to the patient
	err := testData.DataAPI.UpdatePatient(signedupPatientResponse.Patient.ID.Int64(), &api.PatientUpdate{
		ERxID: &erxPatientID,
	}, false)
	if err != nil {
		t.Fatal("Unable to update patient with erx patient id : " + err.Error())
	}
	prescriptionIDForRequestedPrescription := int64(5504)
	testTime := time.Now()
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      12345,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              erxPatientID,
		PatientAddedForRequest:    false,
		RequestDateStamp:          testTime,
		ClinicianID:               clinicianID,
		RequestedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				erx.LexiDrugSynID:     "1234",
				erx.LexiGenProductID:  "12345",
				erx.LexiSynonymTypeID: "123556",
				erx.NDC:               "2415",
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
			DaysSupply:           encoding.NullInt64{},
			PatientInstructions:  "Take once daily",
			OTC:                  false,
			ERx: &common.ERxData{
				DoseSpotClinicianID: clinicianID,
				ErxSentDate:         &testTime,
				PrescriptionID:      encoding.NewObjectID(prescriptionIDForRequestedPrescription),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       123,
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
				Int64Value: 5,
			},
			PatientInstructions: "Take once daily",
			OTC:                 false,
			ERx: &common.ERxData{
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       123,
				ErxSentDate:         &testTime,
				DoseSpotClinicianID: clinicianID,
			},
		},
	}

	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
		Name:         "Walgreens",
		AddressLine1: "116 New Montgomery",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PrescriptionIDToPrescriptionStatuses = map[int64][]common.StatusEvent{
		prescriptionIDForRequestedPrescription: []common.StatusEvent{{Status: api.ERXStatusDeleted}},
	}

	// Call the Consume method
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	test.OK(t, refillRXWorker.Do())

	// There should be an unlinked patient in the patient db
	linkedpatient, err := testData.DataAPI.GetPatientFromErxPatientID(erxPatientID)
	if err != nil {
		t.Fatal("Unable to get patient based on erx patient id to verify the patient information: " + err.Error())
	}

	if linkedpatient.Status != api.PatientRegistered {
		t.Fatal("Patient was expected to be registered but it was not")
	}

	// There should be an unlinked pharmacy treatment in the unlinked_requested_treatment db
	// There should be a dispensed treatment in the pharmacy_dispensed_treatment db
	// There should be a test pharmacy in the pharmacy_selection db
	// There should be a status entry in the refill_request_status table
	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	if len(refillRequestStatuses) != 1 {
		t.Fatal("Expected there to exist 1 refill request status for the refill request just persisted")
	}

	if refillRequestStatuses[0].ItemID != refillRequestItem.ID ||
		refillRequestStatuses[0].Status != api.RXRefillStatusRequested {
		t.Fatal("Refill request status not in expected state")
	}

	// There should be a pending entry in the doctor's queue
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	if err != nil {
		t.Fatal("Unable to get pending items from doctor queue: " + err.Error())
	}

	if len(pendingItems) != 1 {
		t.Fatal("Expected there to exist 1 pending item in the doctor's queue which is the refill request")
	}

	if pendingItems[0].EventType != api.DQEventTypeRefillRequest ||
		pendingItems[0].ItemID != refillRequestStatuses[0].ItemID {
		t.Fatal("Pending item found in the doctor's queue is not the expected item")
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	test.OK(t, err)
	test.Equals(t, false, refillRequest.DispensedPrescription == nil)
	test.Equals(t, false, refillRequest.RequestedPrescription == nil)
	test.Equals(t, int64(0), refillRequest.RequestedPrescription.OriginatingTreatmentID)
	test.Equals(t, false, refillRequest.Patient == nil)
	test.Equals(t, api.PatientRegistered, refillRequest.Patient.Status)
}

// TestRefill_NonSprucePatient is an integration test to ensure that refill requests
// work as expected for non spruce patients.
func TestRefill_NonSprucePatient(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create doctor with clinicianId specicified
	doctor := createDoctorWithClinicianID(testData, t)

	testTime := time.Now()
	// Get StubErx to return refill requests in the refillRequest call
	refillRequestItem := &common.RefillRequestItem{
		RxRequestQueueItemID:      12345,
		ReferenceNumber:           "TestReferenceNumber",
		PharmacyRxReferenceNumber: "TestRxReferenceNumber",
		ErxPatientID:              555,
		PatientAddedForRequest:    true,
		RequestDateStamp:          testTime,
		ClinicianID:               clinicianID,
		RequestedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				erx.LexiDrugSynID:     "1234",
				erx.LexiGenProductID:  "12345",
				erx.LexiSynonymTypeID: "123556",
				erx.NDC:               "2415",
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
				Int64Value: 5,
			}, PatientInstructions: "Take once daily",
			OTC: false,
			ERx: &common.ERxData{
				DoseSpotClinicianID: clinicianID,
				ErxSentDate:         &testTime,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       123,
			},
		},
		DispensedPrescription: &common.Treatment{
			DrugDBIDs: map[string]string{
				erx.LexiDrugSynID:     "1234",
				erx.LexiGenProductID:  "12345",
				erx.LexiSynonymTypeID: "123556",
				erx.NDC:               "2415",
			},
			DrugName:                "Teting (This - Drug)",
			DosageStrength:          "10 mg",
			DispenseValue:           5,
			DispenseUnitDescription: "Tablet",
			DaysSupply: encoding.NullInt64{
				IsValid:    true,
				Int64Value: 5,
			},
			NumberRefills: encoding.NullInt64{
				IsValid:    true,
				Int64Value: 5,
			},
			SubstitutionsAllowed: false,
			PatientInstructions:  "Take once daily",
			OTC:                  false,
			ERx: &common.ERxData{
				DoseSpotClinicianID: clinicianID,
				PrescriptionID:      encoding.NewObjectID(5504),
				PrescriptionStatus:  "Requested",
				ErxPharmacyID:       123,
				ErxSentDate:         &testTime,
			},
		},
	}

	//  Get StubErx to return pharmacy in the GetPharmacyDetails call
	pharmacyToReturn := &pharmacy.PharmacyData{
		SourceID:     1234,
		Source:       pharmacy.PharmacySourceSurescripts,
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
		DOB:          encoding.Date{Year: 2013, Month: 8, Day: 9},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.NewObjectID(12345),
		Pharmacy:     pharmacyToReturn,
	}

	stubErxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubErxAPI.PharmacyDetailsToReturn = pharmacyToReturn
	stubErxAPI.PatientDetailsToReturn = patientToReturn
	stubErxAPI.RefillRxRequestQueueToReturn = []*common.RefillRequestItem{refillRequestItem}

	// Call the Consume method
	refillRXWorker := app_worker.NewRefillRequestWorker(
		testData.DataAPI,
		stubErxAPI,
		&TestLock{},
		testData.Config.Dispatcher,
		testData.Config.MetricsRegistry,
	)
	test.OK(t, refillRXWorker.Do())

	// There should be an unlinked patient in the patient db
	unlinkedPatient, err := testData.DataAPI.GetPatientFromErxPatientID(patientToReturn.ERxPatientID.Int64())
	if err != nil {
		t.Fatal("Unable to get patient based on erx patient id to verify the patient information: " + err.Error())
	}

	if unlinkedPatient.Status != api.PatientUnlinked {
		t.Fatal("Patient was expected to be unlinked but it was not")
	}

	// ensure that the patient has a preferred pharmacy
	if unlinkedPatient.Pharmacy == nil {
		t.Fatalf("Expected patient to have a preferred pharmacy instead it has none")
	}

	if unlinkedPatient.Pharmacy.SourceID != 1234 {
		t.Fatalf("Expected patients preferred pharmacy to have id %s instead it had id %d", "1234", unlinkedPatient.Pharmacy.SourceID)
	}

	// There should be an unlinked pharmacy treatment in the unlinked_requested_treatment db
	// There should be a dispensed treatment in the pharmacy_dispensed_treatment db
	// There should be a test pharmacy in the pharmacy_selection db

	// There should be a status entry in the refill_request_status table
	refillRequestStatuses, err := testData.DataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		t.Fatal("Unable to successfully get the pending refill requests stauses from the db: " + err.Error())
	}

	if len(refillRequestStatuses) != 1 {
		t.Fatal("Expected there to exist 1 refill request status for the refill request just persisted")
	}

	if refillRequestStatuses[0].ItemID != refillRequestItem.ID ||
		refillRequestStatuses[0].Status != api.RXRefillStatusRequested {
		t.Fatal("Refill request status not in expected state")
	}

	// There should be a pending entry in the doctor's queue
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))

	if pendingItems[0].EventType != api.DQEventTypeRefillRequest ||
		pendingItems[0].ItemID != refillRequestStatuses[0].ItemID {
		t.Fatal("Pending item found in the doctor's queue is not the expected item")
	}

	refillRequest, err := testData.DataAPI.GetRefillRequestFromID(refillRequestStatuses[0].ItemID)
	if err != nil {
		t.Fatal("Unable to get refill request that was just added: ", err.Error)
	}

	if refillRequest.DispensedPrescription == nil {
		t.Fatalf("Dispensed prescription was null for the refill request when it shouldn't be")
	}

	if refillRequest.RequestedPrescription == nil {
		t.Fatal("Requested prescription was null for refill request when it shouldn't be")
	}

	if refillRequest.RequestedPrescription.OriginatingTreatmentID != 0 {
		t.Fatal("Requested prescription should be unlinked but was instead found in the system")
	}

	if refillRequest.Patient == nil {
		t.Fatal("Refill request expected to have patient demographics attached to it instead it doesnt")
	}

	if refillRequest.Patient.Status != api.PatientUnlinked {
		t.Fatal("patient should be unlinked but instead it was flagged as registered in our system")
	}

	if refillRequest.RequestedPrescription.Doctor == nil || refillRequest.DispensedPrescription.Doctor == nil {
		t.Fatal("Expected doctor object to be present for the requested and dispensed prescription")
	}

}

func createDoctorWithClinicianID(testData *TestData, t *testing.T) *common.Doctor {
	signedupDoctorResponse, _, _ := SignupRandomTestDoctor(t, testData)
	_, err := testData.DB.Exec(`update doctor set clinician_id = ? where id = ?`, clinicianID, signedupDoctorResponse.DoctorID)
	if err != nil {
		t.Fatal("Unable to assign a clinicianId to the doctor: " + err.Error())
	}

	doctor, err := testData.DataAPI.GetDoctorFromID(signedupDoctorResponse.DoctorID)
	if err != nil {
		t.Fatal("Unable to get doctor based on id: " + err.Error())
	}

	return doctor
}

func approveRefillRequest(refillRequest *common.RefillRequestItem, doctorAccountID int64, comment string, testData *TestData, t *testing.T) {
	requestData := doctorpkg.DoctorRefillRequestRequestData{
		RefillRequestID:      refillRequest.ID,
		Action:               "approve",
		ApprovedRefillAmount: 10,
		Comments:             comment,
	}

	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatal("Unable to marshal json object: " + err.Error())
	}

	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.DoctorRefillRxURLPath, "application/json", bytes.NewReader(jsonData), doctorAccountID)
	if err != nil {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}
	defer resp.Body.Close()

	test.Equals(t, http.StatusOK, resp.StatusCode)
}

func denyRefillRequest(refillRequest *common.RefillRequestItem, doctorAccountID int64, comment string, testData *TestData, t *testing.T) {
	denialReasons, err := testData.DataAPI.GetRefillRequestDenialReasons()
	if err != nil || len(denialReasons) == 0 {
		t.Fatal("Unable to get the denial reasons for the refill request")
	}

	// now, lets go ahead and attempt to deny this refill request
	requestData := doctorpkg.DoctorRefillRequestRequestData{
		RefillRequestID: refillRequest.ID,
		Action:          "deny",
		DenialReasonID:  denialReasons[0].ID,
		Comments:        comment,
	}

	jsonData, err := json.Marshal(requestData)
	test.OK(t, err)
	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.DoctorRefillRxURLPath, "application/json", bytes.NewReader(jsonData), doctorAccountID)
	if err != nil {
		t.Fatal("Unable to make successful request to approve refill request: " + err.Error())
	}
	resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}
