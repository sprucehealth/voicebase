package patient

import (
	"net/http"
	"sort"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/responses"
	"github.com/sprucehealth/backend/libs/httputil"
)

type careTeamHandler struct {
	dataAPI   api.DataAPI
	apiDomain string
}

func NewCareTeamHandler(dataAPI api.DataAPI, apiDomain string) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&careTeamHandler{
				dataAPI:   dataAPI,
				apiDomain: apiDomain,
			}),
			api.RolePatient),
		httputil.Get)
}

type careTeamResponse struct {
	CareTeam []*responses.PatientCareTeamMember `json:"care_team"`
}

func (c *careTeamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientID, err := c.dataAPI.GetPatientIDFromAccountID(apiservice.MustCtxAccount(r.Context()).ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cases, err := c.dataAPI.GetCasesForPatient(patientID, append(common.SubmittedPatientCaseStates(), common.PCStatusOpen.String()))
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	caseIDs := make([]int64, len(cases))
	for i, pc := range cases {
		caseIDs[i] = pc.ID.Int64()
	}

	careTeams, err := c.dataAPI.CaseCareTeams(caseIDs)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	members := make([]*common.CareProviderAssignment, 0)
	memberIDSet := make(map[int64]bool)
	for _, careTeam := range careTeams {
		for _, member := range careTeam.Assignments {
			if memberIDSet[member.ProviderID] {
				continue
			}
			if member.Status == api.StatusActive {
				members = append(members, member)
				memberIDSet[member.ProviderID] = true
			}
		}
	}

	sort.Sort(api.ByCareProviderRole(members))
	resItems := make([]*responses.PatientCareTeamMember, len(members))
	for i, member := range members {
		resItems[i] = responses.TransformCareTeamMember(member, c.apiDomain)
	}

	httputil.JSONResponse(w, http.StatusOK, &careTeamResponse{
		CareTeam: resItems,
	})
}
