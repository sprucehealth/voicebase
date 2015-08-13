package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
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

func NewCareTeamHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&careTeamHandler{
				dataAPI: dataAPI,
			})),
		httputil.Get)
}

func (c *careTeamHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

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

	if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctorID, patientCase.PatientID.Int64(), patientCase.ID.Int64(), c.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (c *careTeamHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	patientCase := requestCache[apiservice.CKPatientCase].(*common.PatientCase)

	assignments, err := c.dataAPI.GetActiveMembersOfCareTeamForCase(patientCase.ID.Int64(), false)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	doctors := make([]*common.Doctor, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment.Status == api.StatusActive {
			switch assignment.ProviderRole {
			case api.RoleDoctor, api.RoleCC:
				doctor, err := c.dataAPI.GetDoctorFromID(assignment.ProviderID)
				if err != nil {
					apiservice.WriteError(ctx, err, w, r)
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
