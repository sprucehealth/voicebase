package doctor_queue

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"
)

type mockDataAPI_itemHandler struct {
	api.DataAPI

	updatesRequested []*api.DoctorQueueUpdate
}

func (m *mockDataAPI_itemHandler) UpdateDoctorQueue(updates []*api.DoctorQueueUpdate) error {
	m.updatesRequested = updates
	return nil
}

func (m *mockDataAPI_itemHandler) GetDoctorFromAccountID(accountID int64) (*common.Doctor, error) {
	return &common.Doctor{
		ID:               encoding.NewObjectID(accountID),
		ShortDisplayName: "CC Name",
	}, nil
}

func TestSuccessfulRemove(t *testing.T) {
	testQueueUpdate(t, http.StatusOK, 1, "CASE_ASSIGNMENT:PENDING:10:100")
	testQueueUpdate(t, http.StatusOK, 1, "CASE_MESSAGE:PENDING:10:100")
	testQueueUpdate(t, http.StatusOK, 2, "PATIENT_VISIT:PENDING:10:100")
	testQueueUpdate(t, http.StatusOK, 2, "PATIENT_VISIT:ONGOING:10:100")
}

func TestUnsuccessfulRemove(t *testing.T) {
	testQueueUpdate(t, http.StatusForbidden, 0, "CASE_ASSIGNMENT:REPLIED:10:100")
}

func testQueueUpdate(t *testing.T, expStatus, expCount int, id string) {
	m := &mockDataAPI_listener{
		patient: &common.Patient{
			FirstName: "First",
			LastName:  "Last",
		},
		doctor: &common.Doctor{
			ID:               encoding.NewObjectID(1),
			ShortDisplayName: "CP Name",
		},
		visit: &common.PatientVisit{
			PatientCaseID: encoding.NewObjectID(1),
		},
	}
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
	if w.Code != expStatus {
		t.Fatalf("Expected %d but got %d [%s]", expStatus, w.Code, golog.Caller(1))
	} else if len(m.updatesRequested) != expCount {
		t.Fatalf("Expected %d but got %d [%s]", expCount, len(m.updatesRequested), golog.Caller(1))
	}
}
