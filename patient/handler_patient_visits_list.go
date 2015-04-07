package patient

import (
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
)

type visitsListHandler struct {
	dataAPI            api.DataAPI
	apiDomain          string
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
	dataAPI api.DataAPI, apiDomain string, dispatcher *dispatch.Dispatcher,
	mediaStore *media.Store, expirationDuration time.Duration,
) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&visitsListHandler{
				dataAPI:            dataAPI,
				apiDomain:          apiDomain,
				dispatcher:         dispatcher,
				mediaStore:         mediaStore,
				expirationDuration: expirationDuration,
			}), []string{"GET"})
}

func (v *visitsListHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.RolePatient {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (v *visitsListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestData := visitsListRequestData{}
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		golog.Errorf(err.Error())
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	var states []string
	if requestData.Completed {
		states = common.SubmittedPatientVisitStates()
		states = append(states, common.TreatedPatientVisitStates()...)
	}
	visits, err := v.dataAPI.GetVisitsForCase(requestData.CaseID, states)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	visitResponses := make([]*PatientVisitResponse, len(visits))
	for i, visit := range visits {

		if visit.Status == common.PVStatusPending {
			if err := checkLayoutVersionForFollowup(v.dataAPI, v.dispatcher, visit, r); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
		intakeInfo, err := IntakeLayoutForVisit(v.dataAPI, v.apiDomain, v.mediaStore, v.expirationDuration, visit)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		submittedDate := visit.SubmittedDate
		visitResponses[i] = &PatientVisitResponse{
			SubmittedDate:   &submittedDate,
			VisitIntakeInfo: intakeInfo,
		}
	}

	httputil.JSONResponse(w, http.StatusOK, visitsListResponse{
		Visits: visitResponses,
	})
}
