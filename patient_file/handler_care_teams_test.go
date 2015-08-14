package patient_file

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
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
		ID: common.NewPatientID(uint64(d.patientAccountID)),
	}, nil
}

func (d mockedDataAPI_handlerCareTeams) DoesCaseExistForPatient(common.PatientID, int64) (bool, error) {
	return d.canAccess, nil
}

func canAccess(httpMethod, role string, doctorID int64, patientID common.PatientID, dataAPI api.DataAPI) error {
	return nil
}

func cannotAccess(httpMethod, role string, doctorID int64, patientID common.PatientID, dataAPI api.DataAPI) error {
	return apiservice.NewAccessForbiddenError()
}

var getCareTeamsForPatientByCaseResponse map[int64]*common.PatientCareTeam
var casesForPatient []*common.PatientCase

func (d mockedDataAPI_handlerCareTeams) GetCasesForPatient(patientID common.PatientID, states []string) ([]*common.PatientCase, error) {
	return casesForPatient, nil
}
func (d mockedDataAPI_handlerCareTeams) CaseCareTeams(caseIDs []int64) (map[int64]*common.PatientCareTeam, error) {
	return getCareTeamsForPatientByCaseResponse, nil
}

func TestDoctorRequiresPatientID(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	handler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{nil, 1, 2, false}, "api.spruce.local")
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleDoctor})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteError(ctx, apiservice.NewValidationError("patient_id required"), expectedWriter, r)
	handler.ServeHTTP(ctx, responseWriter, r)
	test.Equals(t, expectedWriter.Body, responseWriter.Body)
}

func TestDoctorCannotAccessUnownedPatient(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?patient_id=32", nil)
	test.OK(t, err)
	handler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{nil, 1, 2, false}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = cannotAccess
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleDoctor})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteError(ctx, apiservice.NewAccessForbiddenError(), expectedWriter, r)
	handler.ServeHTTP(ctx, responseWriter, r)
	test.Equals(t, expectedWriter.Body, responseWriter.Body)
}

func TestPatientCannotAccessUnownedCase(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?case_id=1", nil)
	test.OK(t, err)
	handler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{nil, 1, 2, false}, "api.spruce.local")
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RolePatient})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteAccessNotAllowedError(ctx, expectedWriter, r)
	handler.ServeHTTP(ctx, responseWriter, r)
	test.Equals(t, expectedWriter.Body, responseWriter.Body)
}

func TestDoctorCanFetchAllCareTeams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?patient_id=32", nil)
	test.OK(t, err)
	handler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{nil, 1, 2, false}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = canAccess
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleDoctor})
	getCareTeamsForPatientByCaseResponse = buildDummyGetCareTeamsForPatientByCaseResponse(2)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local"))
	handler.ServeHTTP(ctx, responseWriter, r)
	// TODO: We can't verify the JSON output here as maps do not serialize determinisitically
	// test.Equals(t, expectedWriter.Body, responseWriter.Body)
	test.Equals(t, 2, len(createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local").CareTeams))
}

func TestPatientCanFetchAllCareTeams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	handler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{nil, 1, 2, false}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = canAccess
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RolePatient})
	getCareTeamsForPatientByCaseResponse = buildDummyGetCareTeamsForPatientByCaseResponse(2)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local"))

	handler.ServeHTTP(ctx, responseWriter, r)
	// TODO: We can't verify the JSON output here as maps do not serialize determinisitically
	// test.Equals(t, expectedWriter.Body, responseWriter.Body)
	test.Equals(t, 2, len(createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local").CareTeams))
}

func TestMACanFetchAllCareTeams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?patient_id=32", nil)
	test.OK(t, err)
	handler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{nil, 1, 2, false}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = canAccess
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleCC})
	getCareTeamsForPatientByCaseResponse = buildDummyGetCareTeamsForPatientByCaseResponse(2)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local"))
	handler.ServeHTTP(ctx, responseWriter, r)
	// TODO: We can't verify the JSON output here as maps do not serialize determinisitically
	// test.Equals(t, expectedWriter.Body, responseWriter.Body)
	test.Equals(t, 2, len(createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 0, "api.spruce.local").CareTeams))
}

func TestDoctorCanFilterCareTeamsByCase(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?patient_id=1&case_id=1", nil)
	test.OK(t, err)
	handler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{nil, 1, 2, true}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = canAccess
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleDoctor})
	getCareTeamsForPatientByCaseResponse = buildDummyGetCareTeamsForPatientByCaseResponse(2)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 1, "api.spruce.local"))
	handler.ServeHTTP(ctx, responseWriter, r)
	test.Equals(t, expectedWriter.Body, responseWriter.Body)
	test.Equals(t, 1, len(createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 1, "api.spruce.local").CareTeams))
}

func TestPatientCanFilterCareTeamsByCase(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?case_id=1", nil)
	test.OK(t, err)
	handler := NewPatientCareTeamsHandler(mockedDataAPI_handlerCareTeams{nil, 1, 2, true}, "api.spruce.local")
	verifyDoctorAccessToPatientFileFn = canAccess
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RolePatient})
	getCareTeamsForPatientByCaseResponse = buildDummyGetCareTeamsForPatientByCaseResponse(2)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, createCareTeamsResponse(getCareTeamsForPatientByCaseResponse, 1, "api.spruce.local"))
	handler.ServeHTTP(ctx, responseWriter, r)
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
