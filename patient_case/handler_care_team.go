package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type careTeamHandler struct {
	dataAPI api.DataAPI
}

type careTeamRequestData struct {
	CaseID int64 `schema:"case_id"`
}

func NewCareTeamHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&careTeamHandler{
			dataAPI: dataAPI,
		}), []string{"GET"})
}

func (c *careTeamHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &careTeamRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	patientCase, err := c.dataAPI.GetPatientCaseFromID(requestData.CaseID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientCase] = patientCase

	doctorID, err := c.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorID, patientCase.PatientID.Int64(), patientCase.ID.Int64(), c.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (c *careTeamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patientCase := ctxt.RequestCache[apiservice.PatientCase].(*common.PatientCase)

	assignments, err := c.dataAPI.GetActiveMembersOfCareTeamForCase(patientCase.ID.Int64(), false)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctors := make([]*common.Doctor, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment.Status == api.StatusActive {
			switch assignment.ProviderRole {
			case api.RoleDoctor, api.RoleMA:
				doctor, err := c.dataAPI.GetDoctorFromID(assignment.ProviderID)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				doctors = append(doctors, doctor)
			}
		}
	}

	httputil.JSONResponse(w, http.StatusOK, struct {
		CareTeam []*common.Doctor `json:"care_team"`
	}{
		CareTeam: doctors,
	})
}
