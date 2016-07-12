package patient_visit

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

type messageHandler struct {
	dataAPI api.DataAPI
}

type messageRequestData struct {
	PatientVisitID int64  `schema:"visit_id" json:"visit_id,string"`
	Message        string `schema:"message" json:"message"`
}

func NewMessageHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&messageHandler{
				dataAPI: dataAPI,
			})),
		httputil.Get, httputil.Put)
}

func (m *messageHandler) IsAuthorized(r *http.Request) (bool, error) {
	account := apiservice.MustCtxAccount(r.Context())
	requestCache := apiservice.MustCtxCache(r.Context())
	if account.Role != api.RolePatient {
		return false, nil
	}

	patientID, err := m.dataAPI.GetPatientIDFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPatientID] = patientID

	requestData := &messageRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = requestData

	patientVisit, err := m.dataAPI.GetPatientVisitFromID(requestData.PatientVisitID)
	if err != nil {
		return false, err
	} else if patientVisit.PatientID != patientID {
		return false, apiservice.NewAccessForbiddenError()
	}
	requestCache[apiservice.CKPatientVisit] = patientVisit

	return true, nil
}

func (m *messageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(r.Context())
	requestData := requestCache[apiservice.CKRequestData].(*messageRequestData)

	switch r.Method {
	case httputil.Get:
		message, err := m.dataAPI.GetMessageForPatientVisit(requestData.PatientVisitID)
		if api.IsErrNotFound(err) {
			apiservice.WriteResourceNotFoundError("message not found", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		httputil.JSONResponse(w, http.StatusOK, struct {
			Message string `json:"message"`
		}{
			Message: message,
		})
	case httputil.Put:
		if err := m.dataAPI.SetMessageForPatientVisit(requestData.PatientVisitID, requestData.Message); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		apiservice.WriteJSONSuccess(w)
	default:
		http.NotFound(w, r)
	}
}
