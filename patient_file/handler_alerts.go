package patient_file

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

type alertsHandler struct {
	dataAPI api.DataAPI
}

func NewAlertsHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.SupportedMethods(
		&alertsHandler{
			dataAPI: dataAPI,
		},
		[]string{apiservice.HTTP_GET},
	)
}

type alertsRequestData struct {
	PatientId int64 `schema:"patient_id"`
}

func (a *alertsHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &alertsRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	switch ctxt.Role {
	case api.DOCTOR_ROLE:
		if requestData.PatientId == 0 {
			return false, apiservice.NewValidationError("patient_id must be specified", r)
		}

		doctorId, err := a.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
		if err != nil {
			return false, err
		}

		if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, doctorId, requestData.PatientId, a.dataAPI); err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientID] = requestData.PatientId
	case api.PATIENT_ROLE:
		patientIdFromAuthToken, err := a.dataAPI.GetPatientIdFromAccountId(ctxt.AccountId)
		if err != nil {
			return false, err
		}

		if requestData.PatientId > 0 {
			if patientIdFromAuthToken != requestData.PatientId {
				return false, apiservice.NewValidationError("patient_id provided does not match the patient making api call", r)
			}
		}
		ctxt.RequestCache[apiservice.PatientID] = patientIdFromAuthToken
	default:
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (a *alertsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patientId := ctxt.RequestCache[apiservice.PatientID].(int64)

	alerts, err := a.dataAPI.GetAlertsForPatient(patientId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{"alerts": alerts})
}
