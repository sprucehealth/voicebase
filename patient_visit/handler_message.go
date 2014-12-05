package patient_visit

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

type messageHandler struct {
	dataAPI api.DataAPI
}

type messageRequestData struct {
	PatientVisitId int64  `schema:"visit_id" json:"visit_id,string"`
	Message        string `schema:"message" json:"message"`
}

func NewMessageHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&messageHandler{
			dataAPI: dataAPI,
		}), []string{"GET", "PUT"})
}

func (m *messageHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.PATIENT_ROLE {
		return false, nil
	}

	patientId, err := m.dataAPI.GetPatientIdFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientID] = patientId

	requestData := &messageRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	patientVisit, err := m.dataAPI.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		return false, err
	} else if patientVisit.PatientId.Int64() != patientId {
		return false, apiservice.NewAccessForbiddenError()
	}
	ctxt.RequestCache[apiservice.PatientVisit] = patientVisit

	return true, nil
}

func (m *messageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*messageRequestData)

	switch r.Method {
	case apiservice.HTTP_GET:
		message, err := m.dataAPI.GetMessageForPatientVisit(requestData.PatientVisitId)
		if err == api.NoRowsError {
			apiservice.WriteResourceNotFoundError("message not found", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		apiservice.WriteJSON(w, map[string]interface{}{
			"message": message,
		})
	case apiservice.HTTP_PUT:
		if err := m.dataAPI.SetMessageForPatientVisit(requestData.PatientVisitId, requestData.Message); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		apiservice.WriteJSONSuccess(w)
	default:
		http.NotFound(w, r)
	}
}
