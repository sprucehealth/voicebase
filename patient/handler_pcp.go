package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/httputil"
)

type pcpHandler struct {
	dataAPI api.DataAPI
}

type pcpData struct {
	PCP *common.PCP `json:"pcp,omitempty"`
}

func NewPCPHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(apiservice.AuthorizationRequired(
		&pcpHandler{
			dataAPI: dataAPI,
		}), []string{"GET", "PUT"})
}

func (p *pcpHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.RolePatient {
		return false, nil
	}
	return true, nil
}

func (p *pcpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		p.getPCP(w, r)
	case httputil.Put:
		p.addPCP(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *pcpHandler) addPCP(w http.ResponseWriter, r *http.Request) {
	requestData := &pcpData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	patientID, err := p.dataAPI.GetPatientIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// if the patient is requesting that the PCP be cleared out, then lets delete
	// all the pcp information
	if requestData.PCP.IsZero() {
		if err := p.dataAPI.DeletePatientPCP(patientID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		apiservice.WriteJSONSuccess(w)
		return
	}

	// validate
	if requestData.PCP.PhysicianName == "" {
		apiservice.WriteValidationError("Please enter primary care physician's name", w, r)
		return
	} else if requestData.PCP.PhoneNumber == "" {
		apiservice.WriteValidationError("Please enter primary care physician's phone number", w, r)
		return
	} else if requestData.PCP.Email != "" && !email.IsValidEmail(requestData.PCP.Email) {
		apiservice.WriteValidationError("Please enter a valid email address", w, r)
		return
	}

	requestData.PCP.PatientID = patientID
	if err := p.dataAPI.UpdatePatientPCP(requestData.PCP); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

func (p *pcpHandler) getPCP(w http.ResponseWriter, r *http.Request) {
	patientID, err := p.dataAPI.GetPatientIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
	}

	pcp, err := p.dataAPI.GetPatientPCP(patientID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, pcpData{PCP: pcp})
}
