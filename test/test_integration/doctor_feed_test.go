package test_integration

import (
	"fmt"
	"github.com/sprucehealth/backend/api"
	"testing"
)

func TestDoctorQueueWithPatientVisits(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// get the current primary doctor
	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)

	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// there should be 1 item in the global queue for the doctor to consume
	elligibleItems, err := testData.DataApi.GetElligibleItemsInUnclaimedQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	}

	elligibleItems, err = testData.DataApi.GetElligibleItemsInUnclaimedQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(elligibleItems) != 6 {
		t.Fatalf("Expected 6 items in the queue instead got %d", len(elligibleItems))
	}

	// now, go ahead and submit the first diagnosis so that it clears from the queue
	SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	elligibleItems, err = testData.DataApi.GetElligibleItemsInUnclaimedQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(elligibleItems) != 5 {
		t.Fatalf("Expected 5 items in the queue but got %d", len(elligibleItems))
	}

	// ensure that there is 1 completed item in the doctor queue
	completedItems, err := testData.DataApi.GetCompletedItemsInDoctorQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(completedItems) != 1 {
		t.Fatalf("Expected 1 item in the completed section but got %d", len(completedItems))
	}
}

func insertIntoDoctorQueue(testData *TestData, doctorQueueItem *api.DoctorQueueItem, t *testing.T) {
	_, err := testData.DB.Exec(fmt.Sprintf(`insert into doctor_queue (doctor_id, event_type, item_id, status) 
												values (?, 'PATIENT_VISIT', ?, '%s')`, doctorQueueItem.Status), doctorQueueItem.DoctorId, doctorQueueItem.ItemId)
	if err != nil {
		t.Fatal("Unable to insert item into doctor queue: " + err.Error())
	}
}

func insertIntoDoctorQueueWithEnqueuedDate(testData *TestData, doctorQueueItem *api.DoctorQueueItem, t *testing.T) {
	_, err := testData.DB.Exec(fmt.Sprintf(`insert into doctor_queue (doctor_id, event_type, item_id, status, enqueue_date) 
												values (?, 'PATIENT_VISIT', ?, '%s', ?)`, doctorQueueItem.Status), doctorQueueItem.DoctorId, doctorQueueItem.ItemId, doctorQueueItem.EnqueueDate)
	if err != nil {
		t.Fatal("Unable to insert item into doctor queue: " + err.Error())
	}
}
