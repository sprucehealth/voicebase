package doctor_queue

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
)

type mockDataAPI_itemHandler struct {
	api.DataAPI

	updatesRequested []*api.DoctorQueueUpdate
}

func (m *mockDataAPI_itemHandler) UpdateDoctorQueue(updates []*api.DoctorQueueUpdate) error {
	m.updatesRequested = updates
	return nil
}

func TestSuccessfulRemove(t *testing.T) {
	testSuccessfulRemove(t, "CASE_ASSIGNMENT:PENDING:10:100")
	testSuccessfulRemove(t, "CASE_MESSAGE:PENDING:10:100")
}

func TestUnsuccessfulRemove(t *testing.T) {
	testForbiddenlRemove(t, "CASE_ASSIGNMENT:REPLIED:10:100")
	testForbiddenlRemove(t, "PATIENT_VISIT:ONGOING:10:100")
	testForbiddenlRemove(t, "PATIENT_VISIT:PENDING:10:100")
}

func testSuccessfulRemove(t *testing.T, id string) {
	m := &mockDataAPI_listener{}
	h := NewItemHandler(m)
	w := httptest.NewRecorder()

	jsonData, err := json.Marshal(itemRequest{
		ID:     id,
		Action: "remove",
	})
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("PUT", "api.spruce.loc", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")

	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, w.Code)
	} else if len(m.updatesRequested) != 1 {
		t.Fatalf("Expected %d but got %d", 1, len(m.updatesRequested))
	}
}

func testForbiddenlRemove(t *testing.T, id string) {
	m := &mockDataAPI_listener{}
	h := NewItemHandler(m)
	w := httptest.NewRecorder()

	jsonData, err := json.Marshal(itemRequest{
		ID:     id,
		Action: "remove",
	})
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("PUT", "api.spruce.loc", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")

	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected %d but got %d", http.StatusForbidden, w.Code)
	} else if len(m.updatesRequested) != 0 {
		t.Fatalf("Expected %d but got %d", 0, len(m.updatesRequested))
	}
}
