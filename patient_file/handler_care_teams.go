package patient_file

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
)

// The base handler struct to handle requests for care team collections related to a patient
type patientCareTeamHandler struct {
	dataAPI   api.DataAPI
	apiDomain string
}

// The request structure expected for use with the handler returned from NewPatientCareTeamHandler
type patientCareTeamRequest struct {
	PatientID int64 `schema:"patient_id"`
	CaseID    int64 `schema:"case_id"`
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
func NewPatientCareTeamsHandler(dataAPI api.DataAPI, apiDomain string) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&patientCareTeamHandler{
					dataAPI:   dataAPI,
					apiDomain: apiDomain,
				}), []string{api.DOCTOR_ROLE, api.PATIENT_ROLE, api.MA_ROLE}), []string{"GET"})
}

// IsAuthorized when given a http.Request object, determines if the caller is authorized to access the needed resources.
// Validation Set:
// 		Doctor:
//			ValidateDoctorAccessToPatientFile
//			DoesCaseExistForPatient
//		Patient:
//			DoesCaseExistForPatient
func (h *patientCareTeamHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	rd := &patientCareTeamRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}

	// Set the patient id either from the params for the user account depending on the user role
	switch ctxt.Role {
	default:
		return false, nil
	case api.DOCTOR_ROLE, api.MA_ROLE:
		if rd.PatientID == 0 {
			return false, apiservice.NewValidationError("patient_id required")
		}

		doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
		if err != nil {
			return false, err
		} else if err := verifyDoctorAccessToPatientFileFn(r.Method, ctxt.Role, doctorID, rd.PatientID, h.dataAPI); err != nil {
			return false, err
		}
	case api.PATIENT_ROLE:
		patient, err := h.dataAPI.GetPatientFromAccountID(ctxt.AccountID)
		if err != nil {
			return false, err
		}
		// Populate the patient id aspect of our request to that it is consumed in a uniform way regardless of user type
		rd.PatientID = patient.PatientID.Int64Value
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

	ctxt.RequestCache[apiservice.RequestData] = rd

	return true, nil
}

// ServeHTTP serves requests to the patientCareTeamHandler
// Utilizes dataAPI.GetCareTeamsForPatient to fetch care teams
// TODO:OPTIMIZATION: This method could be optimized in the way it manages array sizing
// TODO:PAGINATE: This API returns an unbounded list of data and should be paginated in the future
func (h *patientCareTeamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	rd := ctxt.RequestCache[apiservice.RequestData].(*patientCareTeamRequest)

	// get a list of cases for the patient
	careTeams, err := h.dataAPI.GetCareTeamsForPatientByCase(rd.PatientID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, createCareTeamsResponse(careTeams, rd.CaseID, h.apiDomain))
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
