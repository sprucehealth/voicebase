package patient_file

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
)

type mockedDataAPI_handlerCareTeams struct {
	api.DataAPI
	doctorIDFromAccountID int64
	patientAccountID      int64
	canAccess             bool
}

func (d mockedDataAPI_handlerCareTeams) GetDoctorIDFromAccountID(accountID int64) (int64, error) {
	return d.doctorIDFromAccountID, nil
}

func (d mockedDataAPI_handlerCareTeams) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
	return &common.Patient{
		PatientID: encoding.NewObjectID(d.patientAccountID),
	}, nil
}

func (d mockedDataAPI_handlerCareTeams) DoesCaseExistForPatient(p, c int64) (bool, error) {
	return d.canAccess, nil
}

func canAccess(httpMethod, role string, doctorID, patientID int64, dataAPI api.DataAPI) error {
	return nil
}

func cannotAccess(httpMethod, role string, doctorID, patientID int64, dataAPI api.DataAPI) error {
	return apiservice.NewAccessForbiddenError()
}

var getCareTeamsForPatientByCaseResponse map[int64]*common.PatientCareTeam
var casesForPatient []*common.PatientCase

func (d mockedDataAPI_handlerCareTeams) GetCasesForPatient(patientID int64, states []string) ([]*common.PatientCase, error) {
	return casesForPatient, nil
}
func (d mockedDataAPI_handlerCareTeams) CaseCareTeams(caseIDs []int64) (map[int64]*common.PatientCareTeam, error) {
	return getCareTeamsForPatientByCaseResponse, nil
}

func TestDoctorRequiresPatientID(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	careTeamHandler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{&api.DataService{}, 1, 2, false}, "api.spruce.local")
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.RoleDoctor
			ctxt.AccountID = 1
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteError(apiservice.NewValidationError("patient_id required"), expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Body, responseWriter.Body)
}

func TestDoctorCannotAccessUnownedPatient(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?patient_id=32", nil)
	test.OK(t, err)
	careTeamHandler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{&api.DataService{}, 1, 2, false}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = cannotAccess
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.RoleDoctor
			ctxt.AccountID = 1
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteError(apiservice.NewAccessForbiddenError(), expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Body, responseWriter.Body)
}

func TestPatientCannotAccessUnownedCase(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?case_id=1", nil)
	test.OK(t, err)
	careTeamHandler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{&api.DataService{}, 1, 2, false}, "api.spruce.local")
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.RolePatient
			ctxt.AccountID = 1
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteAccessNotAllowedError(expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Body, responseWriter.Body)
}

func TestDoctorCanFetchAllCareTeams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?patient_id=32", nil)
	test.OK(t, err)
	careTeamHandler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{&api.DataService{}, 1, 2, false}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = canAccess
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.RoleDoctor
			ctxt.AccountID = 1
		},
	}
	getCareTeamsForPatientByCaseResponse = buildDummyGetCareTeamsForPatientByCaseResponse(2)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local"))
	handler.ServeHTTP(responseWriter, r)
	// TODO: We can't verify the JSON output here as maps do not serialize determinisitically
	// test.Equals(t, expectedWriter.Body, responseWriter.Body)
	test.Equals(t, 2, len(createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local").CareTeams))
}

func TestPatientCanFetchAllCareTeams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	careTeamHandler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{&api.DataService{}, 1, 2, false}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = canAccess
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.RolePatient
			ctxt.AccountID = 1
		},
	}
	getCareTeamsForPatientByCaseResponse = buildDummyGetCareTeamsForPatientByCaseResponse(2)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local"))

	handler.ServeHTTP(responseWriter, r)
	// TODO: We can't verify the JSON output here as maps do not serialize determinisitically
	// test.Equals(t, expectedWriter.Body, responseWriter.Body)
	test.Equals(t, 2, len(createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local").CareTeams))
}

func TestMACanFetchAllCareTeams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?patient_id=32", nil)
	test.OK(t, err)
	careTeamHandler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{&api.DataService{}, 1, 2, false}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = canAccess
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.RoleCC
			ctxt.AccountID = 1
		},
	}
	getCareTeamsForPatientByCaseResponse = buildDummyGetCareTeamsForPatientByCaseResponse(2)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local"))
	handler.ServeHTTP(responseWriter, r)
	// TODO: We can't verify the JSON output here as maps do not serialize determinisitically
	// test.Equals(t, expectedWriter.Body, responseWriter.Body)
	test.Equals(t, 2, len(createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local").CareTeams))
}

func TestDoctorCanFilterCareTeamsByCase(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?patient_id=1&case_id=1", nil)
	test.OK(t, err)
	careTeamHandler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{&api.DataService{}, 1, 2, true}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = canAccess
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.RoleDoctor
			ctxt.AccountID = 1
		},
	}
	getCareTeamsForPatientByCaseResponse = buildDummyGetCareTeamsForPatientByCaseResponse(2)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 1, "api.spruce.local"))
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Body, responseWriter.Body)
	test.Equals(t, 1, len(createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 1, "api.spruce.local").CareTeams))
}

func TestPatientCanFilterCareTeamsByCase(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?case_id=1", nil)
	test.OK(t, err)
	careTeamHandler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{&api.DataService{}, 1, 2, true}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = canAccess
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.RolePatient
			ctxt.AccountID = 1
		},
	}
	getCareTeamsForPatientByCaseResponse = buildDummyGetCareTeamsForPatientByCaseResponse(2)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 1, "api.spruce.local"))
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Body, responseWriter.Body)
	test.Equals(t, 1, len(createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 1, "api.spruce.local").CareTeams))
}

func buildDummyGetCareTeamsForPatientByCaseResponse(careTeamCount int) map[int64]*common.PatientCareTeam {
	resp := make(map[int64]*common.PatientCareTeam)
	for i := 1; i < careTeamCount+1; i++ {
		team := &common.PatientCareTeam{
			Assignments: make([]*common.CareProviderAssignment, 0),
		}
		assignment := &common.CareProviderAssignment{
			ProviderRole:     "Doctor",
			ProviderID:       1,
			FirstName:        "First",
			LastName:         "Last",
			ShortTitle:       "ShortT",
			LongTitle:        "LongT",
			ShortDisplayName: "SDN",
			LongDisplayName:  "LDN",
			CreationDate:     time.Unix(0, 0),
		}
		team.Assignments = append(team.Assignments, assignment)
		resp[int64(i)] = team
	}
	return resp
}
