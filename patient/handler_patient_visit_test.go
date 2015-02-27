package patient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_PatientVisitHandler struct {
	api.DataAPI
	visit       *common.PatientVisit
	visitUpdate *api.PatientVisitUpdate
	caseUpdate  *api.PatientCaseUpdate
}

func (m *mockDataAPI_PatientVisitHandler) GetPatientVisitFromID(id int64) (*common.PatientVisit, error) {
	return m.visit, nil
}
func (m *mockDataAPI_PatientVisitHandler) UpdatePatientCase(id int64, update *api.PatientCaseUpdate) error {
	m.caseUpdate = update
	return nil
}
func (m *mockDataAPI_PatientVisitHandler) UpdatePatientVisit(id int64, update *api.PatientVisitUpdate) error {
	m.visitUpdate = update
	return nil
}

// This test is to ensure that in the event of a successful
// call to abandon a visit, the appropriate objects are updated
// with the appropriate state.
func TestAbandonVisit_Successful(t *testing.T) {
	m := &mockDataAPI_PatientVisitHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusOpen,
		},
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "api.spruce.local/visit?patient_visit_id=1", nil)
	test.OK(t, err)

	h := NewPatientVisitHandler(m, nil, nil, nil, "", nil, nil, time.Duration(0))
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	test.Equals(t, true, m.caseUpdate != nil)
	test.Equals(t, common.PCStatusDeleted, *m.caseUpdate.Status)
	test.Equals(t, true, m.visitUpdate != nil)
	test.Equals(t, common.PVStatusDeleted, *m.visitUpdate.Status)
}

func TestAbandonVisit_Idempotent(t *testing.T) {
	m := &mockDataAPI_PatientVisitHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusDeleted,
		},
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "api.spruce.local/case?patient_visit_id=1", nil)
	test.OK(t, err)

	h := NewPatientVisitHandler(m, nil, nil, nil, "", nil, nil, time.Duration(0))
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
}

// This test is to ensure that deletion/abandonment of a case in any state other
// than open or deleted is forbidden
func TestAbandonCase_Forbidden(t *testing.T) {
	testForbiddenDelete(t, common.PVStatusRouted)
	testForbiddenDelete(t, common.PVStatusSubmitted)
	testForbiddenDelete(t, common.PVStatusReviewing)
	testForbiddenDelete(t, common.PVStatusTreated)
}

func testForbiddenDelete(t *testing.T, status string) {
	m := &mockDataAPI_PatientVisitHandler{
		visit: &common.PatientVisit{
			Status: status,
		},
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "api.spruce.local/case?patient_visit_id=1", nil)
	test.OK(t, err)

	h := NewPatientVisitHandler(m, nil, nil, nil, "", nil, nil, time.Duration(0))
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusForbidden, w.Code)
}
