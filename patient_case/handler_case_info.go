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

func NewCaseInfoHandler(dataAPI api.DataAPI) *caseInfoHandler {
	return &caseInfoHandler{
		dataAPI: dataAPI,
	}
}

type caseInfoRequestData struct {
	CaseId int64 `schema:"case_id"`
}

type CaseInfoResponseData struct {
	Case *common.PatientCase `json:"case"`
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
	case api.DOCTOR_ROLE:
		doctorId, err := c.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := apiservice.ValidateReadAccessToPatientCase(doctorId, patientCase.PatientId.Int64(), requestData.CaseId, c.dataAPI); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	// get the care team for case
	patientCase.CareTeam, err = c.dataAPI.GetActiveMembersOfCareTeamForCase(requestData.CaseId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &CaseInfoResponseData{Case: patientCase})
}
