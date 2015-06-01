package patient_visit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_PreSubmissionTriageHandler struct {
	api.DataAPI
	visit       *common.PatientVisit
	patientCase *common.PatientCase
	zipCode     string

	visitUpdate *api.PatientVisitUpdate
	caseUpdate  *api.PatientCaseUpdate
}

func (m *mockDataAPI_PreSubmissionTriageHandler) GetPatientVisitFromID(id int64) (*common.PatientVisit, error) {
	return m.visit, nil
}
func (m *mockDataAPI_PreSubmissionTriageHandler) UpdatePatientVisit(id int64, update *api.PatientVisitUpdate) error {
	m.visitUpdate = update
	return nil
}
func (m *mockDataAPI_PreSubmissionTriageHandler) UpdatePatientCase(id int64, update *api.PatientCaseUpdate) error {
	m.caseUpdate = update
	return nil
}
func (m *mockDataAPI_PreSubmissionTriageHandler) GetPatientCaseFromID(id int64) (*common.PatientCase, error) {
	return m.patientCase, nil
}
func (m *mockDataAPI_PreSubmissionTriageHandler) PatientLocation(patientID int64) (string, string, error) {
	return m.zipCode, "", nil
}

func TestVisitTriage_OpenVisit(t *testing.T) {
	m := &mockDataAPI_PreSubmissionTriageHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusOpen,
		},
		patientCase: &common.PatientCase{
			Name: "test",
		},
	}

	dispatcher := dispatch.New()
	h := NewPreSubmissionTriageHandler(m, dispatcher)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("PUT", "api.spruce.local/triage", nil)
	test.OK(t, err)

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
}

func TestVisitTriage_Customize(t *testing.T) {
	m := &mockDataAPI_PreSubmissionTriageHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusOpen,
		},
		patientCase: &common.PatientCase{
			Name: "test",
		},
		zipCode: "94115",
	}

	dispatcher := dispatch.New()
	h := NewPreSubmissionTriageHandler(m, dispatcher)
	w := httptest.NewRecorder()

	expectedActionMessage := "Testing custom action message"
	expectedActionURL := "https://testme.com?zip=<zipcode>"
	expectedTitle := "Testing custom title for triage message"

	jsonData, err := json.Marshal(&presubmissionTriageRequest{
		Title:         expectedTitle,
		ActionMessage: expectedActionMessage,
		ActionURL:     expectedActionURL,
	})
	test.OK(t, err)

	r, err := http.NewRequest("PUT", "api.spruce.local/triage", bytes.NewBuffer(jsonData))
	test.OK(t, err)
	r.Header.Set("Content-Type", "application/json")

	var receivedEvent *PreSubmissionVisitTriageEvent
	dispatcher.Subscribe(func(ev *PreSubmissionVisitTriageEvent) error {
		receivedEvent = ev
		return nil
	})

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, expectedActionMessage, receivedEvent.ActionMessage)
	test.Equals(t, expectedTitle, receivedEvent.Title)
	test.Equals(t, "https://testme.com?zip=94115", receivedEvent.ActionURL)
}

func TestVisitTriage_Abandon(t *testing.T) {
	m := &mockDataAPI_PreSubmissionTriageHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusOpen,
		},
		patientCase: &common.PatientCase{
			Name: "test",
		},
		zipCode: "94115",
	}

	dispatcher := dispatch.New()
	h := NewPreSubmissionTriageHandler(m, dispatcher)
	w := httptest.NewRecorder()

	jsonData, err := json.Marshal(&presubmissionTriageRequest{
		Abandon: true,
	})
	test.OK(t, err)

	r, err := http.NewRequest("PUT", "api.spruce.local/triage", bytes.NewBuffer(jsonData))
	test.OK(t, err)
	r.Header.Set("Content-Type", "application/json")

	h.ServeHTTP(w, r)

	test.Equals(t, true, m.caseUpdate.TimeoutDate.Valid)
	test.Equals(t, common.PCStatusPreSubmissionTriageDeleted, *m.caseUpdate.Status)
}

func TestVisitTriage_TriagedVisit(t *testing.T) {
	m := &mockDataAPI_PreSubmissionTriageHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusPreSubmissionTriage,
		},
		patientCase: &common.PatientCase{
			Name: "test",
		},
		zipCode: "94115",
	}

	expectedTitle := "Your test visit has ended and you should seek medical care today."
	expectedActionURL := "https://www.google.com/?gws_rd=ssl#q=urgent+care+in+94115"
	expectedActionMessage := "How to find a local care provider"

	dispatcher := dispatch.New()
	h := NewPreSubmissionTriageHandler(m, dispatcher)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("PUT", "api.spruce.local/triage", nil)
	test.OK(t, err)

	var receivedEvent *PreSubmissionVisitTriageEvent
	dispatcher.Subscribe(func(ev *PreSubmissionVisitTriageEvent) error {
		receivedEvent = ev
		return nil
	})

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, expectedActionMessage, receivedEvent.ActionMessage)
	test.Equals(t, expectedTitle, receivedEvent.Title)
	test.Equals(t, expectedActionURL, receivedEvent.ActionURL)
	test.Equals(t, true, m.caseUpdate.TimeoutDate.Valid)
	test.Equals(t, common.PCStatusPreSubmissionTriage, *m.caseUpdate.Status)
	test.Equals(t, true, time.Now().Add(23*time.Hour).Before(*m.caseUpdate.TimeoutDate.Time))
}

func TestVisitTriage_SubmittedVisit(t *testing.T) {
	m := &mockDataAPI_PreSubmissionTriageHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusSubmitted,
		},
	}

	dispatcher := &dispatch.Dispatcher{}
	h := NewPreSubmissionTriageHandler(m, dispatcher)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("PUT", "api.spruce.local/triage", nil)
	test.OK(t, err)

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusBadRequest, w.Code)
}
