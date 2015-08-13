package patient_file

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_handlerAlerts struct {
	api.DataAPI
	doctorID        int64
	pc              *common.PatientCase
	visit           *common.PatientVisit
	cases           []*common.PatientCase
	visits          []*common.PatientVisit
	alerts          []*common.Alert
	careTeamsByCase map[int64]*common.PatientCareTeam

	visitIDQueried int64
	caseIDQueried  int64
}

func (m *mockDataAPI_handlerAlerts) GetPatientCaseFromID(caseID int64) (*common.PatientCase, error) {
	return m.pc, nil
}
func (m *mockDataAPI_handlerAlerts) GetPatientVisitFromID(visitID int64) (*common.PatientVisit, error) {
	return m.visit, nil
}
func (m *mockDataAPI_handlerAlerts) GetDoctorIDFromAccountID(accountID int64) (int64, error) {
	return m.doctorID, nil
}
func (m *mockDataAPI_handlerAlerts) CaseCareTeams(caseIDs []int64) (map[int64]*common.PatientCareTeam, error) {
	return m.careTeamsByCase, nil
}
func (m *mockDataAPI_handlerAlerts) GetCasesForPatient(patientID int64, states []string) ([]*common.PatientCase, error) {
	return m.cases, nil
}
func (m *mockDataAPI_handlerAlerts) GetVisitsForCase(caseID int64, states []string) ([]*common.PatientVisit, error) {
	m.caseIDQueried = caseID
	return m.visits, nil
}
func (m *mockDataAPI_handlerAlerts) AlertsForVisit(visitID int64) ([]*common.Alert, error) {
	m.visitIDQueried = visitID
	return m.alerts, nil
}

func TestAlerts_NoParams(t *testing.T) {
	m := &mockDataAPI_handlerAlerts{
		doctorID: 10,
	}

	h := NewAlertsHandler(m)
	w := httptest.NewRecorder()

	r, err := http.NewRequest("GET", "api.spruce.loc/alerts", nil)
	test.OK(t, err)

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleDoctor})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusBadRequest, w.Code)
}

func TestAlerts_ByVisitID(t *testing.T) {
	m := &mockDataAPI_handlerAlerts{
		doctorID: 10,
		careTeamsByCase: map[int64]*common.PatientCareTeam{
			1: &common.PatientCareTeam{
				Assignments: []*common.CareProviderAssignment{
					{
						ProviderID:   10,
						ProviderRole: api.RoleDoctor,
					},
				},
			},
		},
		pc: &common.PatientCase{
			Claimed: true,
		},
		visit: &common.PatientVisit{},
		alerts: []*common.Alert{
			{
				Message: "alert1",
			},
			{
				Message: "alert2",
			},
		},
	}

	h := NewAlertsHandler(m)
	w := httptest.NewRecorder()

	r, err := http.NewRequest("GET", "api.spruce.loc/alerts?patient_visit_id=5&patient_id=10&case_id=11", nil)
	test.OK(t, err)

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleDoctor})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var res alertsResponse
	test.OK(t, json.NewDecoder(w.Body).Decode(&res))
	test.Equals(t, 2, len(res.Alerts))
	test.Equals(t, "alert1", res.Alerts[0].Message)
	test.Equals(t, "alert2", res.Alerts[1].Message)
	test.Equals(t, int64(5), m.visitIDQueried)
}

func TestAlerts_ByCaseID(t *testing.T) {
	m := &mockDataAPI_handlerAlerts{
		doctorID: 10,
		careTeamsByCase: map[int64]*common.PatientCareTeam{
			1: &common.PatientCareTeam{
				Assignments: []*common.CareProviderAssignment{
					{
						ProviderID:   10,
						ProviderRole: api.RoleDoctor,
					},
				},
			},
		},
		pc: &common.PatientCase{
			Claimed: true,
		},
		visit: &common.PatientVisit{},
		visits: []*common.PatientVisit{
			{
				ID:           encoding.NewObjectID(10),
				CreationDate: time.Now().Add(-10 * time.Hour),
			},
			{
				ID:           encoding.NewObjectID(9),
				CreationDate: time.Now().Add(-2 * time.Hour),
			},
		},
		alerts: []*common.Alert{
			{
				Message: "alert1",
			},
			{
				Message: "alert2",
			},
		},
	}

	h := NewAlertsHandler(m)
	w := httptest.NewRecorder()

	r, err := http.NewRequest("GET", "api.spruce.loc/alerts?patient_id=10&case_id=11", nil)
	test.OK(t, err)

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleDoctor})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var res alertsResponse
	test.OK(t, json.NewDecoder(w.Body).Decode(&res))
	test.Equals(t, 2, len(res.Alerts))
	test.Equals(t, "alert1", res.Alerts[0].Message)
	test.Equals(t, "alert2", res.Alerts[1].Message)
	test.Equals(t, int64(9), m.visitIDQueried)
}

func TestAlerts_ByPatientID(t *testing.T) {
	m := &mockDataAPI_handlerAlerts{
		doctorID: 10,
		careTeamsByCase: map[int64]*common.PatientCareTeam{
			1: &common.PatientCareTeam{
				Assignments: []*common.CareProviderAssignment{
					{
						ProviderID:   10,
						ProviderRole: api.RoleDoctor,
					},
				},
			},
		},
		pc: &common.PatientCase{
			Claimed: true,
		},
		visit: &common.PatientVisit{},
		visits: []*common.PatientVisit{
			{
				ID:           encoding.NewObjectID(10),
				CreationDate: time.Now().Add(-10 * time.Hour),
			},
			{
				ID:           encoding.NewObjectID(9),
				CreationDate: time.Now().Add(-2 * time.Hour),
			},
		},
		cases: []*common.PatientCase{
			{
				ID:           encoding.NewObjectID(8),
				CreationDate: time.Now().Add(-10 * time.Hour),
			},
			{
				ID:           encoding.NewObjectID(7),
				CreationDate: time.Now().Add(-2 * time.Hour),
			},
		},
		alerts: []*common.Alert{
			{
				Message: "alert1",
			},
			{
				Message: "alert2",
			},
		},
	}

	h := NewAlertsHandler(m)
	w := httptest.NewRecorder()

	r, err := http.NewRequest("GET", "api.spruce.loc/alerts?patient_id=10", nil)
	test.OK(t, err)

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleDoctor})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var res alertsResponse
	test.OK(t, json.NewDecoder(w.Body).Decode(&res))
	test.Equals(t, 2, len(res.Alerts))
	test.Equals(t, "alert1", res.Alerts[0].Message)
	test.Equals(t, "alert2", res.Alerts[1].Message)
	test.Equals(t, int64(9), m.visitIDQueried)
	test.Equals(t, int64(7), m.caseIDQueried)
}
