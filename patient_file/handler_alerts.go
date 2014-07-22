package patient_file

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

type alertsHandler struct {
	dataAPI api.DataAPI
}

func NewAlertsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		&alertsHandler{
			dataAPI: dataAPI,
		},
		[]string{apiservice.HTTP_GET},
	)
}

type alertsRequestData struct {
	PatientId int64 `schema:"patient_id"`
}

func (a *alertsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestData := &alertsRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	patientId := requestData.PatientId
	ctxt := apiservice.GetContext(r)
	switch ctxt.Role {
	case api.PATIENT_ROLE:
		patientIdFromAuthToken, err := a.dataAPI.GetPatientIdFromAccountId(ctxt.AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if patientId > 0 {
			if patientIdFromAuthToken != patientId {
				apiservice.WriteValidationError("patient_id provided does not match the patient making the api call", w, r)
				return
			}
		} else {
			patientId = requestData.PatientId
		}

	case api.DOCTOR_ROLE:
		if patientId == 0 {
			apiservice.WriteValidationError("patient_id must be specified", w, r)
			return
		}

		doctorId, err := a.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := apiservice.ValidateDoctorAccessToPatientFile(doctorId, patientId, a.dataAPI); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

	default:
		apiservice.WriteAccessNotAllowedError(w, r)
	}

	alerts, err := a.dataAPI.GetAlertsForPatient(patientId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{"alerts": alerts})
}
