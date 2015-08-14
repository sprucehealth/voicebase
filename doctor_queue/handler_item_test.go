package doctor_queue

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
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
		ID:               encoding.DeprecatedNewObjectID(accountID),
		ShortDisplayName: "CC Name",
	}, nil
}

func TestSuccessfulRemove(t *testing.T) {
	testQueueUpdate(t, http.StatusOK, 1, "CASE_ASSIGNMENT:PENDING:10:100:doctor")
	testQueueUpdate(t, http.StatusOK, 1, "CASE_MESSAGE:PENDING:10:100:doctor")
	testQueueUpdate(t, http.StatusOK, 2, "PATIENT_VISIT:PENDING:10:100:unclaimed")
	testQueueUpdate(t, http.StatusOK, 2, "PATIENT_VISIT:ONGOING:10:100:doctor")
}

func TestUnsuccessfulRemove(t *testing.T) {
	testQueueUpdate(t, http.StatusForbidden, 0, "CASE_ASSIGNMENT:REPLIED:10:100:doctor")
}

func testQueueUpdate(t *testing.T, expStatus, expCount int, id string) {
	m := &mockDataAPI_listener{
		patient: &common.Patient{
			FirstName: "First",
			LastName:  "Last",
		},
		doctor: &common.Doctor{
			ID:               encoding.DeprecatedNewObjectID(1),
			ShortDisplayName: "CP Name",
		},
		visit: &common.PatientVisit{
			PatientCaseID: encoding.DeprecatedNewObjectID(1),
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

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RoleCC, ID: 1})
	h.ServeHTTP(ctx, w, r)
	if w.Code != expStatus {
		t.Fatalf("Expected %d but got %d [%s]", expStatus, w.Code, golog.Caller(1))
	} else if len(m.updatesRequested) != expCount {
		t.Fatalf("Expected %d but got %d [%s]", expCount, len(m.updatesRequested), golog.Caller(1))
	}
	if expStatus == http.StatusOK {
		exp := api.DoctorQueueType(id[strings.LastIndex(id, ":")+1:])
		if v := m.updatesRequested[0].QueueItem.QueueType; v != exp {
			t.Fatalf("Expected '%s' got '%s' for queue type", exp, v)
		}
	}
}
