package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
)

type caseInfoHandler struct {
	dataAPI   api.DataAPI
	apiDomain string
}

func NewCaseInfoHandler(dataAPI api.DataAPI, apiDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			apiservice.SupportedRoles(
				&caseInfoHandler{
					dataAPI:   dataAPI,
					apiDomain: apiDomain,
				}, api.RolePatient, api.RoleDoctor)),
		httputil.Get)
}

type caseInfoRequestData struct {
	CaseID int64 `schema:"case_id"`
}

type caseInfoResponseData struct {
	Case       *responses.Case `json:"case"`
	CaseConfig struct {
		MessagingEnabled            bool   `json:"messaging_enabled"`
		MessagingDisabledReason     string `json:"messaging_disabled_reason"`
		TreatmentPlanEnabled        bool   `json:"treatment_plan_enabled"`
		TreatmentPlanDisabledReason string `json:"treatment_plan_disabled_reason"`
	} `json:"case_config"`
}

func (c *caseInfoHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestData := &caseInfoRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	} else if requestData.CaseID == 0 {
		apiservice.WriteValidationError(ctx, "case_id must be specified", w, r)
		return
	}

	patientCase, err := c.dataAPI.GetPatientCaseFromID(requestData.CaseID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	responseData := &caseInfoResponseData{}

	account := apiservice.MustCtxAccount(ctx)
	switch account.Role {
	case api.RolePatient:
		patientID, err := c.dataAPI.GetPatientIDFromAccountID(account.ID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		// ensure that the case is owned by the patient
		if patientID != patientCase.PatientID {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}

		// messaging is enabled even if the case is not claimed as the patient should be able to message with the MA
		// at any time
		responseData.CaseConfig.MessagingEnabled = true

		// treatment plan is enabled if one exists
		activeTreatmentPlanExists, err := c.dataAPI.DoesActiveTreatmentPlanForCaseExist(patientCase.ID.Int64())
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		if !activeTreatmentPlanExists {
			responseData.CaseConfig.TreatmentPlanDisabledReason = "Your doctor will create a treatment plan just for you."
		} else {
			responseData.CaseConfig.TreatmentPlanEnabled = true
		}

	case api.RoleDoctor:
		doctorID, err := c.dataAPI.GetDoctorIDFromAccountID(account.ID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctorID, patientCase.PatientID, requestData.CaseID, c.dataAPI); err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	patientVisits, err := c.dataAPI.GetVisitsForCase(patientCase.ID.Int64(), nil)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// set the case level diagnosis to be that of the latest treated patient visit
	var diagnosis string
	for _, visit := range patientVisits {
		if visit.Status == common.PVStatusTreated {
			diagnosis, err = c.dataAPI.DiagnosisForVisit(visit.ID.Int64())
			if !api.IsErrNotFound(err) && err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}
			break
		}
	}

	if patientCase.Status == common.PCStatusUnsuitable {
		diagnosis = "Unsuitable for Spruce"
	} else if diagnosis == "" {
		diagnosis = "Pending"
	}

	// only set the care team if the patient has been claimed or the case has been marked as unsuitable
	var careTeamMembers []*responses.PatientCareTeamMember
	if patientCase.Claimed {
		// get the care team for case
		members, err := c.dataAPI.GetActiveMembersOfCareTeamForCase(requestData.CaseID, true)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		careTeamMembers = make([]*responses.PatientCareTeamMember, len(members))
		for i, member := range members {
			careTeamMembers[i] = responses.TransformCareTeamMember(member, c.apiDomain)
		}
	}
	responseData.Case = responses.NewCase(patientCase, careTeamMembers, diagnosis)

	httputil.JSONResponse(w, http.StatusOK, &responseData)
}
