package doctor_queue

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type mockDataAPI_DoctorQueue struct {
	api.DataAPI
	inboxItems      []*api.DoctorQueueItem
	ccInboxItems    []*api.DoctorQueueItem
	unassignedItems []*api.DoctorQueueItem
	historyItems    []*api.DoctorQueueItem
	patients        map[common.PatientID]*common.Patient
}

func (m *mockDataAPI_DoctorQueue) GetDoctorIDFromAccountID(accountID int64) (int64, error) {
	return 0, nil
}

func (m *mockDataAPI_DoctorQueue) GetPendingItemsInDoctorQueue(doctorID int64) ([]*api.DoctorQueueItem, error) {
	return m.inboxItems, nil
}

func (m *mockDataAPI_DoctorQueue) GetPendingItemsInCCQueues() ([]*api.DoctorQueueItem, error) {
	return m.ccInboxItems, nil
}

func (m *mockDataAPI_DoctorQueue) GetPendingItemsForClinic() ([]*api.DoctorQueueItem, error) {
	return m.unassignedItems, nil
}

func (m *mockDataAPI_DoctorQueue) GetElligibleItemsInUnclaimedQueue(doctorID int64) ([]*api.DoctorQueueItem, error) {
	return m.unassignedItems, nil
}

func (m *mockDataAPI_DoctorQueue) GetCompletedItemsForClinic() ([]*api.DoctorQueueItem, error) {
	return m.historyItems, nil
}

func (m *mockDataAPI_DoctorQueue) GetCompletedItemsInDoctorQueue(doctorID int64) ([]*api.DoctorQueueItem, error) {
	return m.historyItems, nil
}

func (m *mockDataAPI_DoctorQueue) Patients([]common.PatientID) (map[common.PatientID]*common.Patient, error) {
	return m.patients, nil
}

func TestInbox_CC(t *testing.T) {
	m := &mockDataAPI_DoctorQueue{}

	m.ccInboxItems = []*api.DoctorQueueItem{
		{
			Description: "Testing",
			Tags:        []string{"test"},
			PatientID:   common.NewPatientID(1),
		},
	}
	m.patients = map[common.PatientID]*common.Patient{
		common.NewPatientID(1): {
			FirstName: "kunal",
			LastName:  "jham",
		},
	}

	h := NewInboxHandler(m)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local/inbox", nil)
	test.OK(t, err)

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RoleCC})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var res struct {
		Items []*DoctorQueueDisplayItem `json:"items"`
	}

	test.OK(t, json.Unmarshal(w.Body.Bytes(), &res))
	test.Equals(t, 1, len(res.Items))
	test.Equals(t, 1, len(res.Items[0].Tags))
	test.Equals(t, "test", res.Items[0].Tags[0])
}

func TestInbox_Tags(t *testing.T) {
	m := &mockDataAPI_DoctorQueue{
		inboxItems: []*api.DoctorQueueItem{
			{
				Description: "Testing",
				Tags:        []string{"test"},
				PatientID:   common.NewPatientID(1),
			},
		},
		patients: map[common.PatientID]*common.Patient{
			common.NewPatientID(1): {
				FirstName: "kunal",
				LastName:  "jham",
			},
		},
	}

	h := NewInboxHandler(m)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local/inbox", nil)
	test.OK(t, err)

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RoleDoctor})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var res struct {
		Items []*DoctorQueueDisplayItem `json:"items"`
	}

	test.OK(t, json.Unmarshal(w.Body.Bytes(), &res))
	test.Equals(t, 1, len(res.Items))
	test.Equals(t, 1, len(res.Items[0].Tags))
	test.Equals(t, "test", res.Items[0].Tags[0])
}

func TestUnassigned_Tags(t *testing.T) {
	m := &mockDataAPI_DoctorQueue{}

	m.unassignedItems = []*api.DoctorQueueItem{
		{
			Description: "Testing",
			Tags:        []string{"test"},
			PatientID:   common.NewPatientID(1),
		},
	}
	m.patients = map[common.PatientID]*common.Patient{
		common.NewPatientID(1): {
			FirstName: "kunal",
			LastName:  "jham",
		},
	}

	h := NewUnassignedHandler(m)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local/unassigned", nil)
	test.OK(t, err)

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RoleCC})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var res struct {
		Items []*DoctorQueueDisplayItem `json:"items"`
	}

	test.OK(t, json.Unmarshal(w.Body.Bytes(), &res))
	test.Equals(t, 1, len(res.Items))
	test.Equals(t, 1, len(res.Items[0].Tags))
	test.Equals(t, "test", res.Items[0].Tags[0])
}

func TestCompleted_Tags(t *testing.T) {
	m := &mockDataAPI_DoctorQueue{}

	m.historyItems = []*api.DoctorQueueItem{
		{
			Description: "Testing",
			Tags:        []string{"test"},
			PatientID:   common.NewPatientID(1),
		},
	}
	m.patients = map[common.PatientID]*common.Patient{
		common.NewPatientID(1): {
			FirstName: "kunal",
			LastName:  "jham",
		},
	}

	h := NewHistoryHandler(m)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local/history", nil)
	test.OK(t, err)

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RoleCC})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var res struct {
		Items []*DoctorQueueDisplayItem `json:"items"`
	}

	test.OK(t, json.Unmarshal(w.Body.Bytes(), &res))
	test.Equals(t, 1, len(res.Items))
	test.Equals(t, 1, len(res.Items[0].Tags))
	test.Equals(t, "test", res.Items[0].Tags[0])
}
