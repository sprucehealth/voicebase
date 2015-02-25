package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
)

type careTeamHandler struct {
	dataAPI   api.DataAPI
	apiDomain string
}

func NewCareTeamHandler(dataAPI api.DataAPI, apiDomain string) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&careTeamHandler{
			dataAPI:   dataAPI,
			apiDomain: apiDomain,
		}), []string{"GET"})
}

type careTeamResponse struct {
	CareTeam []*responses.PatientCareTeamMember `json:"care_team"`
}

func (c *careTeamHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.PATIENT_ROLE {
		return false, nil
	}

	return true, nil
}

func (c *careTeamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientID, err := c.dataAPI.GetPatientIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	careTeam, err := c.dataAPI.GetActiveMembersOfCareTeamForPatient(patientID, true)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	members := make([]*responses.PatientCareTeamMember, len(careTeam))
	for i, careTeamMember := range careTeam {
		members[i] = responses.TransformCareTeamMember(careTeamMember, c.apiDomain)
	}

	httputil.JSONResponse(w, http.StatusOK, &careTeamResponse{
		CareTeam: members,
	})
}
