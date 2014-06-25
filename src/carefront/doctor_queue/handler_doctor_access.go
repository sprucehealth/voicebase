package doctor_queue

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"net/http"
)

type grantPatientFileAccessHandler struct {
	dataAPI api.DataAPI
}

type GrantPatientFileRequestData struct {
	PatientCaseId encoding.ObjectId `json:"case_id"`
}

func NewGrantPatientFileAccessHandler(dataAPI api.DataAPI) *grantPatientFileAccessHandler {
	return &grantPatientFileAccessHandler{
		dataAPI: dataAPI,
	}
}

func (g *grantPatientFileAccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

	requestData := GrantPatientFileRequestData{}
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError("Unable to parse input parameters", w, r)
		return
	} else if requestData.PatientCaseId.Int64() == 0 {
		apiservice.WriteValidationError("case_id must be specified", w, r)
		return
	}

	doctorId, err := g.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patientCase, err := g.dataAPI.GetPatientCaseFromId(requestData.PatientCaseId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	err = apiservice.ValidateWriteAccessToPatientCase(doctorId, patientCase.PatientId.Int64(), patientCase.Id.Int64(), g.dataAPI)
	if err == nil {
		// doctor already has access, in which case we return success
		apiservice.WriteJSONSuccess(w)
		return
	}

	switch err.(type) {
	case apiservice.AccessForbiddenError:
		// this means that the doctor does not have permissions yet,
		// in which case this doctor can be granted access to the case
	default:
		apiservice.WriteError(err, w, r)
		return
	}

	// to grant access to the patient case, patient case has to be in unclaimed state
	if patientCase.Status != common.PCStatusUnclaimed {
		apiservice.WriteValidationError("Expected patient case to be in the unclaimed state but it wasnt", w, r)
		return
	} else if err := g.dataAPI.TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient(doctorId, patientCase.Id.Int64(),
		patientCase.PatientId.Int64(), ExpireDuration); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteError(err, w, r)
}
