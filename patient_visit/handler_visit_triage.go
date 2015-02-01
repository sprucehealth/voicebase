package patient_visit

import (
	"net/http"
	"sync"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type presubmissionTriageHandler struct {
	dataAPI api.DataAPI
}

type presubmissionTriageRequest struct {
	PatientVisitID int64 `json:"patient_visit_id,string"`
}

func NewPreSubmissionTriageHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&presubmissionTriageHandler{
					dataAPI: dataAPI,
				}), []string{api.PATIENT_ROLE}), []string{"PUT"})
}

func (p *presubmissionTriageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rd presubmissionTriageRequest
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// ensure that the visit is either in an open state or a pre-submission triaged state
	visit, err := p.dataAPI.GetPatientVisitFromID(rd.PatientVisitID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if !(visit.Status == common.PVStatusPreSubmissionTriage || visit.Status == common.PVStatusOpen) {
		apiservice.WriteValidationError("only an open visit can under pre-submission triage", w, r)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)

	go func() {
		defer wg.Done()

		// update the patient visit status
		now := time.Now()
		updatedStatus := common.PVStatusPreSubmissionTriage
		if err := p.dataAPI.UpdatePatientVisit(rd.PatientVisitID, &api.PatientVisitUpdate{
			ClosedDate: &now,
			Status:     &updatedStatus,
		}); err != nil {
			errs <- err
		}
	}()

	go func() {
		defer wg.Done()

		updatedStatus := common.PCStatusPreSubmissionTriage
		now := time.Now()
		if err := p.dataAPI.UpdatePatientCase(visit.PatientCaseID.Int64(), &api.PatientCaseUpdate{
			Status:     &updatedStatus,
			ClosedDate: &now,
		}); err != nil {
			errs <- err
		}
	}()

	select {
	case err := <-errs:
		apiservice.WriteError(err, w, r)
		return
	default:
	}

	wg.Wait()
	apiservice.WriteJSONSuccess(w)
}
