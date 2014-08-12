package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type caseInfoHandler struct {
	dataAPI api.DataAPI
}

func NewCaseInfoHandler(dataAPI api.DataAPI) http.Handler {
	return &caseInfoHandler{
		dataAPI: dataAPI,
	}
}

type caseInfoRequestData struct {
	CaseId int64 `schema:"case_id"`
}

type caseInfoResponseData struct {
	Case       *common.PatientCase `json:"case"`
	CaseConfig struct {
		MessagingEnabled            bool   `json:"messaging_enabled"`
		MessagingDisabledReason     string `json:"messaging_disabled_reason"`
		TreatmentPlanEnabled        bool   `json:"treatment_plan_enabled"`
		TreatmentPlanDisabledReason string `json:"treatment_plan_disabled_reason"`
	} `json:"case_config"`
}

func (c *caseInfoHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

func (c *caseInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

	requestData := &caseInfoRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	} else if requestData.CaseId == 0 {
		apiservice.WriteValidationError("case_id must be specified", w, r)
		return
	}

	patientCase, err := c.dataAPI.GetPatientCaseFromId(requestData.CaseId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	responseData := &caseInfoResponseData{}

	ctxt := apiservice.GetContext(r)
	switch ctxt.Role {
	case api.PATIENT_ROLE:
		patientId, err := c.dataAPI.GetPatientIdFromAccountId(ctxt.AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// ensure that the case is owned by the patient
		if patientId != patientCase.PatientId.Int64() {
			apiservice.WriteAccessNotAllowedError(w, r)
			return
		}

		// messaging is enabled even if the case is not claimed as the patient should be able to message with the MA
		// at any time
		responseData.CaseConfig.MessagingEnabled = true

		// treatment plan is enabled if one exists
		activeTreatmentPlanExists, err := c.dataAPI.DoesActiveTreatmentPlanForCaseExist(patientCase.Id.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if !activeTreatmentPlanExists {
			responseData.CaseConfig.TreatmentPlanDisabledReason = "Check back soon for your custom treatment plan."
		} else {
			responseData.CaseConfig.TreatmentPlanEnabled = true
		}

	case api.DOCTOR_ROLE:
		doctorId, err := c.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorId, patientCase.PatientId.Int64(), requestData.CaseId, c.dataAPI); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	// only set the diagnosis for the patient case if the latest visit has been treated
	// FIX: Populating the diagnosis for a case based on the latest visit's diagnosis for now.
	// Need to figure out how to store the diagnosis at the case level.
	patientVisits, err := c.dataAPI.GetVisitsForCase(patientCase.Id.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if patientVisits[0].Status == common.PVStatusTreated {
		patientCase.Diagnosis = patientVisits[0].Diagnosis
	}

	// only set the care team if the patient has been claimed
	if patientCase.Status == common.PCStatusClaimed {
		// get the care team for case
		patientCase.CareTeam, err = c.dataAPI.GetActiveMembersOfCareTeamForCase(requestData.CaseId, true)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}
	responseData.Case = patientCase

	apiservice.WriteJSON(w, &responseData)
}
