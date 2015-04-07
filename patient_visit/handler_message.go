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
	PatientVisitID int64  `schema:"visit_id" json:"visit_id,string"`
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
	if ctxt.Role != api.RolePatient {
		return false, nil
	}

	patientID, err := m.dataAPI.GetPatientIDFromAccountID(ctxt.AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientID] = patientID

	requestData := &messageRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	patientVisit, err := m.dataAPI.GetPatientVisitFromID(requestData.PatientVisitID)
	if err != nil {
		return false, err
	} else if patientVisit.PatientID.Int64() != patientID {
		return false, apiservice.NewAccessForbiddenError()
	}
	ctxt.RequestCache[apiservice.PatientVisit] = patientVisit

	return true, nil
}

func (m *messageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*messageRequestData)

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
