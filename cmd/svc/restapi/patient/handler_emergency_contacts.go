package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type emergencyContactsHandler struct {
	dataAPI api.DataAPI
}

type emergencyContactsData struct {
	EmergencyContacts []*common.EmergencyContact `json:"emergency_contacts,omitempty"`
}

func NewEmergencyContactsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&emergencyContactsHandler{
					dataAPI: dataAPI,
				}),
			api.RolePatient),
		httputil.Get, httputil.Put)
}

func (e *emergencyContactsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		e.getEmergencyContacts(w, r)
	case httputil.Put:
		e.addEmergencyContacts(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (e *emergencyContactsHandler) getEmergencyContacts(w http.ResponseWriter, r *http.Request) {
	patientID, err := e.dataAPI.GetPatientIDFromAccountID(apiservice.MustCtxAccount(r.Context()).ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	emergencyContacts, err := e.dataAPI.GetPatientEmergencyContacts(patientID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &emergencyContactsData{EmergencyContacts: emergencyContacts})
}

func (e *emergencyContactsHandler) addEmergencyContacts(w http.ResponseWriter, r *http.Request) {
	requestData := &emergencyContactsData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// validate
	for _, eContact := range requestData.EmergencyContacts {
		if eContact.FullName == "" {
			apiservice.WriteValidationError("Please enter emergency contact's name", w, r)
			return
		} else if eContact.PhoneNumber == "" {
			apiservice.WriteValidationError("Please enter emergency contact's phone number", w, r)
			return
		}
	}

	patientID, err := e.dataAPI.GetPatientIDFromAccountID(apiservice.MustCtxAccount(r.Context()).ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := e.dataAPI.UpdatePatientEmergencyContacts(patientID, requestData.EmergencyContacts); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, requestData)
}
