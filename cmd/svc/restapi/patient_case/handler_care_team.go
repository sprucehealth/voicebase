package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
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
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&careTeamHandler{
				dataAPI: dataAPI,
			})),
		httputil.Get)
}

func (c *careTeamHandler) IsAuthorized(r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(r.Context())
	account := apiservice.MustCtxAccount(r.Context())

	requestData := &careTeamRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = requestData

	patientCase, err := c.dataAPI.GetPatientCaseFromID(requestData.CaseID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPatientCase] = patientCase

	doctorID, err := c.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctorID] = doctorID

	if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctorID, patientCase.PatientID, patientCase.ID.Int64(), c.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (c *careTeamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(r.Context())
	patientCase := requestCache[apiservice.CKPatientCase].(*common.PatientCase)

	assignments, err := c.dataAPI.GetActiveMembersOfCareTeamForCase(patientCase.ID.Int64(), false)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctors := make([]*common.Doctor, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment.Status == api.StatusActive {
			switch assignment.ProviderRole {
			case api.RoleDoctor, api.RoleCC:
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
