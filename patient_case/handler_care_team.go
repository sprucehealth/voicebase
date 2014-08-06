package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type careTeamHandler struct {
	dataAPI api.DataAPI
}

type careTeamRequestData struct {
	CaseID int64 `schema:"case_id"`
}

func NewCareTeamHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.SupportedMethods(&careTeamHandler{
		dataAPI: dataAPI,
	}, []string{apiservice.HTTP_GET})
}

func (c *careTeamHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &careTeamRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	patientCase, err := c.dataAPI.GetPatientCaseFromId(requestData.CaseID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientCase] = patientCase

	doctorId, err := c.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorId

	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorId, patientCase.PatientId.Int64(), patientCase.Id.Int64(), c.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (c *careTeamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patientCase := ctxt.RequestCache[apiservice.PatientCase].(*common.PatientCase)

	assignments, err := c.dataAPI.GetActiveMembersOfCareTeamForCase(patientCase.Id.Int64(), false)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctors := make([]*common.Doctor, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment.Status == api.STATUS_ACTIVE {
			switch assignment.ProviderRole {
			case api.DOCTOR_ROLE, api.MA_ROLE:
				doctor, err := c.dataAPI.GetDoctorFromId(assignment.ProviderID)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				doctors = append(doctors, doctor)
			}
		}
	}

	apiservice.WriteJSON(w, &map[string]interface{}{
		"care_team": doctors,
	})
}
