package patient_visit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_PreSubmissionTriageHandler struct {
	api.DataAPI
	visit *common.PatientVisit
}

func (m *mockDataAPI_PreSubmissionTriageHandler) GetPatientVisitFromID(id int64) (*common.PatientVisit, error) {
	return m.visit, nil
}
func (m *mockDataAPI_PreSubmissionTriageHandler) UpdatePatientVisit(id int64, update *api.PatientVisitUpdate) error {
	return nil
}
func (m *mockDataAPI_PreSubmissionTriageHandler) UpdatePatientCase(id int64, update *api.PatientCaseUpdate) error {
	return nil
}

func TestVisitTriage_OpenVisit(t *testing.T) {
	m := &mockDataAPI_PreSubmissionTriageHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusOpen,
		},
	}

	h := NewPreSubmissionTriageHandler(m)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("PUT", "api.spruce.local/triage", nil)
	test.OK(t, err)

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
}

func TestVisitTriage_TriagedVisit(t *testing.T) {
	m := &mockDataAPI_PreSubmissionTriageHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusPreSubmissionTriage,
		},
	}

	h := NewPreSubmissionTriageHandler(m)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("PUT", "api.spruce.local/triage", nil)
	test.OK(t, err)

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
}

func TestVisitTriage_SubmittedVisit(t *testing.T) {
	m := &mockDataAPI_PreSubmissionTriageHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusSubmitted,
		},
	}

	h := NewPreSubmissionTriageHandler(m)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("PUT", "api.spruce.local/triage", nil)
	test.OK(t, err)

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusBadRequest, w.Code)
}
