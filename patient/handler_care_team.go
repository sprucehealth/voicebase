package patient

import (
	"net/http"
	"sort"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
)

type careTeamHandler struct {
	dataAPI   api.DataAPI
	apiDomain string
}

func NewCareTeamHandler(dataAPI api.DataAPI, apiDomain string) httputil.ContextHandler {
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

func (c *careTeamHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	patientID, err := c.dataAPI.GetPatientIDFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	cases, err := c.dataAPI.GetCasesForPatient(patientID, append(common.SubmittedPatientCaseStates(), common.PCStatusOpen.String()))
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	caseIDs := make([]int64, len(cases))
	for i, pc := range cases {
		caseIDs[i] = pc.ID.Int64()
	}

	careTeams, err := c.dataAPI.CaseCareTeams(caseIDs)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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
