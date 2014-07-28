package doctor_queue

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

// claimPatientCaseAccessHandler is an API handler to grant temporary access to a patient file
// for a doctor to claim the patient case
type claimPatientCaseAccessHandler struct {
	dataAPI          api.DataAPI
	tempClaimSuccess metrics.Counter
	tempClaimFailure metrics.Counter
}

type ClaimPatientCaseRequestData struct {
	PatientCaseId encoding.ObjectId `json:"case_id"`
}

func NewClaimPatientCaseAccessHandler(dataAPI api.DataAPI, statsRegistry metrics.Registry) *claimPatientCaseAccessHandler {
	tempClaimSuccess := metrics.NewCounter()
	tempClaimFailure := metrics.NewCounter()

	statsRegistry.Add("temp_claim/success", tempClaimSuccess)
	statsRegistry.Add("temp_claim/failure", tempClaimFailure)

	return &claimPatientCaseAccessHandler{
		dataAPI:          dataAPI,
		tempClaimSuccess: tempClaimSuccess,
		tempClaimFailure: tempClaimFailure,
	}
}

func (c *claimPatientCaseAccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

	// only the doctor is authorized to claim the ase
	if apiservice.GetContext(r).Role != api.DOCTOR_ROLE {
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}

	requestData := ClaimPatientCaseRequestData{}
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError("Unable to parse input parameters", w, r)
		return
	} else if requestData.PatientCaseId.Int64() == 0 {
		apiservice.WriteValidationError("case_id must be specified", w, r)
		return
	}

	doctorId, err := c.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patientCase, err := c.dataAPI.GetPatientCaseFromId(requestData.PatientCaseId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	err = apiservice.ValidateWriteAccessToPatientCase(doctorId, patientCase.PatientId.Int64(), patientCase.Id.Int64(), c.dataAPI)
	if err == nil {
		// doctor already has access, in which case we return success
		apiservice.WriteJSONSuccess(w)
		return
	}

	switch err {
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
	} else if err := c.dataAPI.TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient(doctorId, patientCase, ExpireDuration); err != nil {
		c.tempClaimFailure.Inc(1)
		apiservice.WriteError(err, w, r)
		return
	}

	c.tempClaimSuccess.Inc(1)
	apiservice.WriteJSONSuccess(w)
}
