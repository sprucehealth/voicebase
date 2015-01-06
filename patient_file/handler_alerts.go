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
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&alertsHandler{
					dataAPI: dataAPI,
				}), []string{api.DOCTOR_ROLE, api.PATIENT_ROLE}),
		[]string{"GET"})

}

type alertsRequestData struct {
	PatientID int64 `schema:"patient_id"`
}

func (a *alertsHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &alertsRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	switch ctxt.Role {
	case api.DOCTOR_ROLE:
		if requestData.PatientID == 0 {
			return false, apiservice.NewValidationError("patient_id must be specified")
		}

		doctorID, err := a.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
		if err != nil {
			return false, err
		}

		if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, doctorID, requestData.PatientID, a.dataAPI); err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientID] = requestData.PatientID
	case api.PATIENT_ROLE:
		patientIdFromAuthToken, err := a.dataAPI.GetPatientIDFromAccountID(ctxt.AccountID)
		if err != nil {
			return false, err
		}

		if requestData.PatientID > 0 {
			if patientIdFromAuthToken != requestData.PatientID {
				return false, apiservice.NewValidationError("patient_id provided does not match the patient making api call")
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
	patientID := ctxt.RequestCache[apiservice.PatientID].(int64)

	alerts, err := a.dataAPI.GetAlertsForPatient(patientID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{"alerts": alerts})
}
