package patient_file

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/responses"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

// The base handler struct to handle requests for care team collections related to a patient
type patientCareTeamHandler struct {
	dataAPI   api.DataAPI
	apiDomain string
}

// The request structure expected for use with the handler returned from NewPatientCareTeamHandler
type patientCareTeamRequest struct {
	PatientID common.PatientID `schema:"patient_id"`
	CaseID    int64            `schema:"case_id"`
}

// The response for requests services by the handler returned from NewPatientCareTeamHandler
type PatientCareTeamResponse struct {
	CareTeams []*responses.PatientCareTeamSummary `json:"care_teams"` // Provides a list of care team summaries containing the related case_id
}

func (r PatientCareTeamResponse) String() string {
	return fmt.Sprintf("{CareTeams: %v}", r.CareTeams)
}

// TODO:REFACTOR: Since the apiservice package doesn't use an interface to implement this fn we need to manage it locally as
//		a *fn for it to be correctly stubbed. Refactoring this aspect of the apiservice package to be more test friendly would be good.
var verifyDoctorAccessToPatientFileFn = apiservice.ValidateDoctorAccessToPatientFile

// NewPatientCareTeamsHandler returns a new handler to access the care teams associated with a given patient.
// Authorization Required: true
// Supported Roles: DOCTOR_ROLE, MA_ROLE, PATIENT_ROLE
func NewPatientCareTeamsHandler(dataAPI api.DataAPI, apiDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(
					&patientCareTeamHandler{
						dataAPI:   dataAPI,
						apiDomain: apiDomain,
					})),
			api.RoleDoctor, api.RolePatient, api.RoleCC),
		httputil.Get)
}

// IsAuthorized when given a http.Request object, determines if the caller is authorized to access the needed resources.
// Validation Set:
// 		Doctor:
//			ValidateDoctorAccessToPatientFile
//			DoesCaseExistForPatient
//		Patient:
//			DoesCaseExistForPatient
func (h *patientCareTeamHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	account := apiservice.MustCtxAccount(ctx)
	requestCache := apiservice.MustCtxCache(ctx)

	rd := &patientCareTeamRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}

	// Set the patient id either from the params for the user account depending on the user role
	switch account.Role {
	default:
		return false, nil
	case api.RoleDoctor, api.RoleCC:
		if !rd.PatientID.IsValid {
			return false, apiservice.NewValidationError("patient_id required")
		}

		doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(account.ID)
		if err != nil {
			return false, err
		} else if err := verifyDoctorAccessToPatientFileFn(r.Method, account.Role, doctorID, rd.PatientID, h.dataAPI); err != nil {
			return false, err
		}
	case api.RolePatient:
		patient, err := h.dataAPI.GetPatientFromAccountID(account.ID)
		if err != nil {
			return false, err
		}
		// Populate the patient id aspect of our request to that it is consumed in a uniform way regardless of user type
		rd.PatientID = patient.ID
	}

	// If we have requested a case we don't have access to, throw an error
	if rd.CaseID != 0 {
		ok, err := h.dataAPI.DoesCaseExistForPatient(rd.PatientID, rd.CaseID)
		if err != nil {
			return false, err
		} else if !ok {
			return false, nil
		}
	}

	requestCache[apiservice.CKRequestData] = rd

	return true, nil
}

// ServeHTTP serves requests to the patientCareTeamHandler
// Utilizes dataAPI.GetCareTeamsForPatient to fetch care teams
// TODO:OPTIMIZATION: This method could be optimized in the way it manages array sizing
// TODO:PAGINATE: This API returns an unbounded list of data and should be paginated in the future
func (h *patientCareTeamHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	rd := requestCache[apiservice.CKRequestData].(*patientCareTeamRequest)

	// get a list of cases for the patient
	cases, err := h.dataAPI.GetCasesForPatient(rd.PatientID, append(common.SubmittedPatientCaseStates(), common.PCStatusOpen.String()))
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	caseIDs := make([]int64, len(cases))
	for i, pc := range cases {
		caseIDs[i] = pc.ID.Int64()
	}

	careTeams, err := h.dataAPI.CaseCareTeams(caseIDs)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, createCareTeamsResponse(careTeams, rd.CaseID, h.apiDomain))
}

// createCareTeamsByCaseToCareTeamsResponse translates (and filters if needed) a map of care teams by case into a care teams response.
func createCareTeamsResponse(careTeamsByCase map[int64]*common.PatientCareTeam, requestedCaseID int64, apiDomain string) PatientCareTeamResponse {
	careTeamResponse := PatientCareTeamResponse{CareTeams: make([]*responses.PatientCareTeamSummary, 0, len(careTeamsByCase))}
	for patientCaseID, careTeam := range careTeamsByCase {
		// Filter by the requested case id if one was provided
		if requestedCaseID != 0 && requestedCaseID != patientCaseID {
			continue
		}

		// Initialize our summary with an empty list and the correct case information
		careTeamSummary := &responses.PatientCareTeamSummary{
			CaseID:  patientCaseID,
			Members: make([]*responses.PatientCareTeamMember, len(careTeam.Assignments)),
		}

		// Translate our DB representations into the client friendly versions
		for i, assignment := range careTeam.Assignments {
			careTeamSummary.Members[i] = responses.TransformCareTeamMember(assignment, apiDomain)
		}

		careTeamResponse.CareTeams = append(careTeamResponse.CareTeams, careTeamSummary)
	}
	return careTeamResponse
}