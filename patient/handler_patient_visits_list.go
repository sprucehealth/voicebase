package patient

import (
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/storage"
)

type visitsListHandler struct {
	dataAPI            api.DataAPI
	dispatcher         *dispatch.Dispatcher
	store              storage.Store
	expirationDuration time.Duration
}

type visitsListRequestData struct {
	CaseID    int64 `schema:"case_id,required"`
	Completed bool  `schema:"completed"`
}

type visitsListResponse struct {
	Visits []*PatientVisitResponse `json:"visits"`
}

func NewVisitsListHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, store storage.Store, expirationDuration time.Duration) http.Handler {
	return &visitsListHandler{
		dataAPI:            dataAPI,
		dispatcher:         dispatcher,
		store:              store,
		expirationDuration: expirationDuration,
	}
}

func (v *visitsListHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	if r.Method != apiservice.HTTP_GET {
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
		clientLayout, err := GetPatientVisitLayout(v.dataAPI, v.dispatcher, v.store, v.expirationDuration, visit, r)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		submittedDate := visit.SubmittedDate
		visitResponses[i] = &PatientVisitResponse{
			SubmittedDate:  &submittedDate,
			PatientVisitId: visit.PatientVisitId.Int64(),
			Status:         visit.Status,
			ClientLayout:   clientLayout,
		}
	}

	apiservice.WriteJSON(w, visitsListResponse{
		Visits: visitResponses,
	})
}
