package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type emergencyContactsHandler struct {
	dataAPI api.DataAPI
}

type emergencyContactsData struct {
	EmergencyContacts []*common.EmergencyContact `json:"emergency_contacts,omitempty"`
}

func NewEmergencyContactsHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&emergencyContactsHandler{
					dataAPI: dataAPI,
				}),
			api.RolePatient),
		httputil.Get, httputil.Put)
}

func (e *emergencyContactsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		e.getEmergencyContacts(ctx, w, r)
	case httputil.Put:
		e.addEmergencyContacts(ctx, w, r)
	default:
		http.NotFound(w, r)
	}
}

func (e *emergencyContactsHandler) getEmergencyContacts(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	patientID, err := e.dataAPI.GetPatientIDFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	emergencyContacts, err := e.dataAPI.GetPatientEmergencyContacts(patientID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &emergencyContactsData{EmergencyContacts: emergencyContacts})
}

func (e *emergencyContactsHandler) addEmergencyContacts(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestData := &emergencyContactsData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	// validate
	for _, eContact := range requestData.EmergencyContacts {
		if eContact.FullName == "" {
			apiservice.WriteValidationError(ctx, "Please enter emergency contact's name", w, r)
			return
		} else if eContact.PhoneNumber == "" {
			apiservice.WriteValidationError(ctx, "Please enter emergency contact's phone number", w, r)
			return
		}
	}

	patientID, err := e.dataAPI.GetPatientIDFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if err := e.dataAPI.UpdatePatientEmergencyContacts(patientID, requestData.EmergencyContacts); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, requestData)
}
