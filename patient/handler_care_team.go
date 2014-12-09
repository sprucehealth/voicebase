package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

type careTeamHandler struct {
	dataAPI api.DataAPI
}

func NewCareTeamHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&careTeamHandler{
			dataAPI: dataAPI,
		}), []string{"GET"})
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

	apiservice.WriteJSON(w, map[string]interface{}{
		"care_team": careTeam,
	})
}
