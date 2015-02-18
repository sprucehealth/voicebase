package patient_case

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_CaseHandler struct {
	api.DataAPI
	patientCase *common.PatientCase
	visits      []*common.PatientVisit
	visitUpdate *api.PatientVisitUpdate
	caseUpdate  *api.PatientCaseUpdate
}

func (m *mockDataAPI_CaseHandler) GetPatientCaseFromID(caseID int64) (*common.PatientCase, error) {
	return m.patientCase, nil
}
func (m *mockDataAPI_CaseHandler) GetVisitsForCase(caseID int64, states []string) ([]*common.PatientVisit, error) {
	return m.visits, nil
}
func (m *mockDataAPI_CaseHandler) UpdatePatientCase(id int64, update *api.PatientCaseUpdate) error {
	m.caseUpdate = update
	return nil
}
func (m *mockDataAPI_CaseHandler) UpdatePatientVisit(id int64, update *api.PatientVisitUpdate) error {
	m.visitUpdate = update
	return nil
}

// This test is to ensure that in the event of a successful
// call to abandon a visit, the appropriate objects are updated
// with the appropriate state.
func TestAbandonCase_Successful(t *testing.T) {
	m := &mockDataAPI_CaseHandler{
		patientCase: &common.PatientCase{
			Status: common.PCStatusOpen,
		},
		visits: []*common.PatientVisit{
			{
				Status: common.PVStatusOpen,
			},
		},
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "api.spruce.local/case?case_id=1", nil)
	test.OK(t, err)

	h := NewHandler(m)
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	test.Equals(t, true, m.caseUpdate != nil)
	test.Equals(t, common.PCStatusDeleted, *m.caseUpdate.Status)
	test.Equals(t, true, m.visitUpdate != nil)
	test.Equals(t, common.PVStatusDeleted, *m.visitUpdate.Status)
}

func TestAbandonCase_Idempotent(t *testing.T) {
	m := &mockDataAPI_CaseHandler{
		patientCase: &common.PatientCase{
			Status: common.PCStatusDeleted,
		},
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "api.spruce.local/case?case_id=1", nil)
	test.OK(t, err)

	h := NewHandler(m)
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
}

// This test is to ensure that deletion/abandonment of a case in any state other
// than open or deleted is forbidden
func TestAbandonCase_Forbidden(t *testing.T) {
	testForbiddenDelete(t, common.PCStatusActive)
	testForbiddenDelete(t, common.PCStatusInactive)
	testForbiddenDelete(t, common.PCStatusPreSubmissionTriage)
}

func testForbiddenDelete(t *testing.T, status common.CaseStatus) {
	m := &mockDataAPI_CaseHandler{
		patientCase: &common.PatientCase{
			Status: status,
		},
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "api.spruce.local/case?case_id=1", nil)
	test.OK(t, err)

	h := NewHandler(m)
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusForbidden, w.Code)
}
