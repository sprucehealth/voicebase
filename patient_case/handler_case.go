package patient_case

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type caseHandler struct {
	dataAPI api.DataAPI
}

type caseRequest struct {
	caseID int64 `schema:"case_id,required"`
}

func NewHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&caseHandler{
				dataAPI: dataAPI,
			}), []string{api.PATIENT_ROLE}),
		[]string{"DELETE"})
}

func (c *caseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rd caseRequest
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	patientCase, err := c.dataAPI.GetPatientCaseFromID(rd.caseID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if patientCase.Status == common.PCStatusDeleted {
		apiservice.WriteJSONSuccess(w)
		return
	} else if patientCase.Status != common.PCStatusOpen {
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}

	visits, err := c.dataAPI.GetVisitsForCase(rd.caseID, common.OpenPatientVisitStates())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if len(visits) != 1 {
		apiservice.WriteError(fmt.Errorf("Expected a single visit for the case but got %d", len(visits)), w, r)
		return
	}

	// update the visit to mark it as deleted
	visitStatus := common.PVStatusDeleted
	if err := c.dataAPI.UpdatePatientVisit(visits[0].PatientVisitID.Int64(), &api.PatientVisitUpdate{
		Status: &visitStatus,
	}); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// update the case to mark it as deleted
	caseStatus := common.PCStatusDeleted
	if err := c.dataAPI.UpdatePatientCase(rd.caseID, &api.PatientCaseUpdate{
		Status: &caseStatus,
	}); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
