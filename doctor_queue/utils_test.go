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
	}

	id := constructIDFromItem(queueItem)
	expectedID := "PATIENT_VISIT:PENDING:100:10"
	if id != expectedID {
		t.Fatalf("Expected: %s Got: %s", expectedID, id)
	}
}

func TestIDBreakdown(t *testing.T) {
	eventType, status, itemID, doctorID, err := queueItemPartsFromID("PATIENT_VISIT:PENDING:100:10")
	if err != nil {
		t.Fatalf(err.Error())
	} else if eventType != api.DQEventTypePatientVisit {
		t.Fatalf("Expected %s got %s", api.DQEventTypePatientVisit, eventType)
	} else if status != api.DQItemStatusPending {
		t.Fatalf("Expected %s got %s", api.DQItemStatusPending, status)
	} else if itemID != 100 {
		t.Fatalf("Expected %d got %d", 100, itemID)
	} else if doctorID != 10 {
		t.Fatalf("Expected %d got %d", 10, doctorID)
	}
}
