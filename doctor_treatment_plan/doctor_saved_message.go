package doctor_treatment_plan

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

type savedMessageHandler struct {
	dataAPI api.DataAPI
}

type DoctorSavedMessageRequestData struct {
	DoctorID        int64  `json:"doctor_id"`
	TreatmentPlanID int64  `json:"treatment_plan_id,string" schema:"treatment_plan_id"`
	Message         string `json:"message"`
}

type doctorSavedMessageGetResponse struct {
	Message string `json:"message"`
}

func NewSavedMessageHandler(dataAPI api.DataAPI) http.Handler {
	return &savedMessageHandler{
		dataAPI: dataAPI,
	}
}

func (h *savedMessageHandler) IsAuthorized(r *http.Request) (bool, error) {
	switch apiservice.GetContext(r).Role {
	case api.DOCTOR_ROLE, api.ADMIN_ROLE:
	default:
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (h *savedMessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)
	var doctorID int64
	switch ctx.Role {
	case api.DOCTOR_ROLE:
		var err error
		doctorID, err = h.dataAPI.GetDoctorIdFromAccountId(ctx.AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	case api.ADMIN_ROLE:
		// The doctor_id will be parsed in the get/put handlers
	default:
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case apiservice.HTTP_GET:
		h.get(w, r, doctorID, ctx)
	case apiservice.HTTP_PUT:
		h.put(w, r, doctorID, ctx)
	default:
		http.NotFound(w, r)
	}
}

func (h *savedMessageHandler) get(w http.ResponseWriter, r *http.Request, doctorID int64, ctx *apiservice.Context) {
	if doctorID == 0 {
		// Admin access
		var err error
		doctorID, err = strconv.ParseInt(r.FormValue("doctor_id"), 10, 64)
		if err != nil {
			apiservice.WriteValidationError("doctor_id is required", w, r)
			return
		}
	}

	requestData := &DoctorSavedMessageRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// Retrieve treatment plan message if it exists. Otherwise, retrieve the default message
	var msg string
	var err error
	if requestData.TreatmentPlanID != 0 {
		msg, err = h.dataAPI.GetTreatmentPlanMessageForDoctor(doctorID, requestData.TreatmentPlanID)
	}
	if err == api.NoRowsError || requestData.TreatmentPlanID == 0 {
		msg, err = h.dataAPI.GetSavedMessageForDoctor(doctorID)
	}

	if err == api.NoRowsError {
		msg = ""
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &doctorSavedMessageGetResponse{Message: msg})
}

func (h *savedMessageHandler) put(w http.ResponseWriter, r *http.Request, doctorID int64, ctx *apiservice.Context) {
	var req DoctorSavedMessageRequestData
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}
	if doctorID == 0 {
		// Admin access
		doctorID = req.DoctorID
		if doctorID == 0 {
			apiservice.WriteValidationError("doctor_id is required", w, r)
			return
		}
	}

	if req.TreatmentPlanID == 0 {
		// Set doctor's standard response
		if err := h.dataAPI.SetSavedMessageForDoctor(doctorID, req.Message); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	} else {
		// Update message for a treatment plan
		if err := h.dataAPI.SetTreatmentPlanMessage(doctorID, req.TreatmentPlanID, req.Message); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSONSuccess(w)
}
