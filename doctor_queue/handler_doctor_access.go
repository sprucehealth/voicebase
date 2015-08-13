package doctor_queue

import (
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
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

func NewClaimPatientCaseAccessHandler(dataAPI api.DataAPI, analyticsLogger analytics.Logger, statsRegistry metrics.Registry) httputil.ContextHandler {
	tempClaimSuccess := metrics.NewCounter()
	tempClaimFailure := metrics.NewCounter()

	statsRegistry.Add("temp_claim/success", tempClaimSuccess)
	statsRegistry.Add("temp_claim/failure", tempClaimFailure)

	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&claimPatientCaseAccessHandler{
				dataAPI:          dataAPI,
				analyticsLogger:  analyticsLogger,
				tempClaimSuccess: tempClaimSuccess,
				tempClaimFailure: tempClaimFailure,
			}), api.RoleDoctor), httputil.Post)
}

func (c *claimPatientCaseAccessHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)

	// only the doctor is authorized to claim the ase
	if account.Role != api.RoleDoctor {
		apiservice.WriteAccessNotAllowedError(ctx, w, r)
		return
	}

	requestData := ClaimPatientCaseRequestData{}
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(ctx, "Unable to parse input parameters", w, r)
		return
	} else if requestData.PatientCaseID.Int64() == 0 {
		apiservice.WriteValidationError(ctx, "case_id must be specified", w, r)
		return
	}

	doctorID, err := c.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	patientCase, err := c.dataAPI.GetPatientCaseFromID(requestData.PatientCaseID.Int64())
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	err = apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctorID, patientCase.PatientID.Int64(), patientCase.ID.Int64(), c.dataAPI)
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
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// to grant access to the patient case, patient case has to be in unclaimed state
	if patientCase.Claimed {
		apiservice.WriteAccessNotAllowedError(ctx, w, r)
		return
	} else if err := c.dataAPI.TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient(doctorID, patientCase, ExpireDuration); err != nil {
		c.tempClaimFailure.Inc(1)
		apiservice.WriteError(ctx, err, w, r)
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
