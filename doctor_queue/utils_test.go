package doctor_queue

import (
	"testing"

	"github.com/sprucehealth/backend/api"
)

func TestIDCreation(t *testing.T) {
	queueItem := &api.DoctorQueueItem{
		EventType: api.DQEventTypePatientVisit,
		Status:    api.DQItemStatusPending,
		DoctorID:  10,
		ItemID:    100,
		QueueType: api.DQTUnclaimedQueue,
	}

	id := constructIDFromItem(queueItem)
	expectedID := "PATIENT_VISIT:PENDING:100:10:unclaimed"
	if id != expectedID {
		t.Fatalf("Expected: %s Got: %s", expectedID, id)
	}
}

func TestIDBreakdown(t *testing.T) {
	qid, err := queueItemPartsFromID("PATIENT_VISIT:PENDING:100:10:unclaimed")
	if err != nil {
		t.Fatalf(err.Error())
	} else if qid.eventType != api.DQEventTypePatientVisit {
		t.Fatalf("Expected %s got %s", api.DQEventTypePatientVisit, qid.eventType)
	} else if qid.status != api.DQItemStatusPending {
		t.Fatalf("Expected %s got %s", api.DQItemStatusPending, qid.status)
	} else if qid.itemID != 100 {
		t.Fatalf("Expected %d got %d", 100, qid.itemID)
	} else if qid.doctorID != 10 {
		t.Fatalf("Expected %d got %d", 10, qid.doctorID)
	} else if qid.queueType != api.DQTUnclaimedQueue {
		t.Fatalf("Expected %d got %d", api.DQTUnclaimedQueue, qid.queueType)
	}
}
