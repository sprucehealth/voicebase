package test_integration

import (
	"fmt"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
)

func TestDoctorQueueWithPatientVisits(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// get the current primary doctor
	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)

	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}
	dc := DoctorClient(testData, t, doctorID)
	pv := CreateRandomPatientVisitInState("CA", t, testData)

	// there should be 1 item in the global queue for the doctor to consume
	elligibleItems, err := testData.DataAPI.GetElligibleItemsInUnclaimedQueue(doctor.ID.Int64())
	test.OK(t, err)

	for i := 0; i < 5; i++ {
		CreateRandomPatientVisitInState("CA", t, testData)
	}

	elligibleItems, err = testData.DataAPI.GetElligibleItemsInUnclaimedQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 6, len(elligibleItems))

	// now, go ahead and start TP for the case
	pc, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)

	test.OK(t, dc.ClaimCase(pc.ID.Int64()))
	_, err = dc.ReviewVisit(pv.PatientVisitID)
	test.OK(t, err)

	tp, err := dc.PickTreatmentPlanForVisit(pv.PatientVisitID, nil)
	if err != nil {
		t.Fatal(err.Error())
	} else if err := dc.UpdateTreatmentPlanNote(tp.ID.Int64(), "foo"); err != nil {
		t.Fatalf(err.Error())
	} else if err := dc.SubmitTreatmentPlan(tp.ID.Int64()); err != nil {
		t.Fatal(err.Error())
	}

	elligibleItems, err = testData.DataAPI.GetElligibleItemsInUnclaimedQueue(doctor.ID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(elligibleItems) != 5 {
		t.Fatalf("Expected 5 items in the queue but got %d", len(elligibleItems))
	}

	// ensure that there is 1 completed item in the doctor queue
	completedItems, err := testData.DataAPI.GetCompletedItemsInDoctorQueue(doctor.ID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(completedItems) != 1 {
		t.Fatalf("Expected 1 item in the completed section but got %d", len(completedItems))
	}
}

func insertIntoDoctorQueue(testData *TestData, doctorQueueItem *api.DoctorQueueItem, t *testing.T) {
	_, err := testData.DB.Exec(fmt.Sprintf(`insert into doctor_queue (doctor_id, event_type, item_id, status) 
												values (?, 'PATIENT_VISIT', ?, '%s')`, doctorQueueItem.Status), doctorQueueItem.DoctorID, doctorQueueItem.ItemID)
	if err != nil {
		t.Fatal("Unable to insert item into doctor queue: " + err.Error())
	}
}

func insertIntoDoctorQueueWithEnqueuedDate(testData *TestData, doctorQueueItem *api.DoctorQueueItem, t *testing.T) {
	_, err := testData.DB.Exec(fmt.Sprintf(`insert into doctor_queue (doctor_id, event_type, item_id, status, enqueue_date) 
												values (?, 'PATIENT_VISIT', ?, '%s', ?)`, doctorQueueItem.Status), doctorQueueItem.DoctorID, doctorQueueItem.ItemID, doctorQueueItem.EnqueueDate)
	if err != nil {
		t.Fatal("Unable to insert item into doctor queue: " + err.Error())
	}
}
