package doctor_queue

import (
	"net/http"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
)

// claimPatientCaseAccessHandler is an API handler to grant temporary access to a patient file
// for a doctor to claim the patient case
type claimPatientCaseAccessHandler struct {
	dataAPI          api.DataAPI
	analyticsLogger  analytics.Logger
	tempClaimSuccess *metrics.Counter
	tempClaimFailure *metrics.Counter
}

type ClaimPatientCaseRequestData struct {
	PatientCaseID encoding.ObjectID `json:"case_id"`
}

func NewClaimPatientCaseAccessHandler(dataAPI api.DataAPI, analyticsLogger analytics.Logger, statsRegistry metrics.Registry) http.Handler {
	tempClaimSuccess := metrics.NewCounter()
	tempClaimFailure := metrics.NewCounter()

	statsRegistry.Add("temp_claim/success", tempClaimSuccess)
	statsRegistry.Add("temp_claim/failure", tempClaimFailure)

	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&claimPatientCaseAccessHandler{
			dataAPI:          dataAPI,
			analyticsLogger:  analyticsLogger,
			tempClaimSuccess: tempClaimSuccess,
			tempClaimFailure: tempClaimFailure,
		}), httputil.Post)
}

func (c *claimPatientCaseAccessHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.RoleDoctor {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (c *claimPatientCaseAccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)

	// only the doctor is authorized to claim the ase
	if ctxt.Role != api.RoleDoctor {
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}

	requestData := ClaimPatientCaseRequestData{}
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError("Unable to parse input parameters", w, r)
		return
	} else if requestData.PatientCaseID.Int64() == 0 {
		apiservice.WriteValidationError("case_id must be specified", w, r)
		return
	}

	doctorID, err := c.dataAPI.GetDoctorIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patientCase, err := c.dataAPI.GetPatientCaseFromID(requestData.PatientCaseID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	err = apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorID, patientCase.PatientID.Int64(), patientCase.ID.Int64(), c.dataAPI)
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
	if patientCase.Claimed {
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	} else if err := c.dataAPI.TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient(doctorID, patientCase, ExpireDuration); err != nil {
		c.tempClaimFailure.Inc(1)
		apiservice.WriteError(err, w, r)
		return
	}

	go func() {
		c.analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "jbcq_temp_assign",
				Timestamp: analytics.Time(time.Now()),
				DoctorID:  doctorID,
				CaseID:    patientCase.ID.Int64(),
			},
		})
	}()

	c.tempClaimSuccess.Inc(1)
	apiservice.WriteJSONSuccess(w)
}
