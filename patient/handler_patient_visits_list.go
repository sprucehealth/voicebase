package patient

import (
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
)

type visitsListHandler struct {
	dataAPI            api.DataAPI
	apiDomain          string
	webDomain          string
	dispatcher         *dispatch.Dispatcher
	mediaStore         *media.Store
	expirationDuration time.Duration
}

type visitsListRequestData struct {
	CaseID    int64 `schema:"case_id,required"`
	Completed bool  `schema:"completed"`
}

type visitsListResponse struct {
	Visits []*PatientVisitResponse `json:"visits"`
}

func NewVisitsListHandler(
	dataAPI api.DataAPI, apiDomain, webDomain string, dispatcher *dispatch.Dispatcher,
	mediaStore *media.Store, expirationDuration time.Duration,
) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&visitsListHandler{
					dataAPI:            dataAPI,
					apiDomain:          apiDomain,
					webDomain:          webDomain,
					dispatcher:         dispatcher,
					mediaStore:         mediaStore,
					expirationDuration: expirationDuration,
				}), api.RolePatient), httputil.Get)
}

func (v *visitsListHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var requestData visitsListRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	patient, err := v.dataAPI.GetPatientFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	pcase, err := v.dataAPI.GetPatientCaseFromID(requestData.CaseID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// Make sure case is owned by patient
	if patient.ID.Int64() != pcase.PatientID.Int64() {
		apiservice.WriteAccessNotAllowedError(ctx, w, r)
		return
	}

	var states []string
	if requestData.Completed {
		states = append(common.SubmittedPatientVisitStates(), common.TreatedPatientVisitStates()...)
	}
	visits, err := v.dataAPI.GetVisitsForCase(requestData.CaseID, states)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	visitResponses := make([]*PatientVisitResponse, len(visits))
	for i, visit := range visits {

		if visit.Status == common.PVStatusPending {
			if err := checkLayoutVersionForFollowup(v.dataAPI, v.dispatcher, visit, r); err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}
		}
		intakeInfo, err := IntakeLayoutForVisit(v.dataAPI, v.apiDomain, v.webDomain, v.mediaStore, v.expirationDuration, visit, patient, api.RolePatient)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		submittedDate := visit.SubmittedDate
		visitResponses[i] = &PatientVisitResponse{
			SubmittedDate:      &submittedDate,
			SubmittedTimestamp: submittedDate.Unix(),
			VisitIntakeInfo:    intakeInfo,
		}
	}

	httputil.JSONResponse(w, http.StatusOK, visitsListResponse{
		Visits: visitResponses,
	})
}
