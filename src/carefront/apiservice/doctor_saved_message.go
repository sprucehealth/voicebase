package apiservice

import (
	"carefront/api"
	"net/http"
	"strconv"
)

type doctorSavedMessageHandler struct {
	dataAPI api.DataAPI
}

type doctorSavedMessagePutRequest struct {
	DoctorID int64  `json:"doctor_id"` // for admin use
	Message  string `json:"message"`
}

type doctorSavedMessageGetResponse struct {
	Message string `json:"message"`
}

func NewDoctorSavedMessageHandler(dataAPI api.DataAPI) http.Handler {
	return &doctorSavedMessageHandler{
		dataAPI: dataAPI,
	}
}

func (h *doctorSavedMessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := GetContext(r)
	var doctorID int64
	switch ctx.Role {
	case api.DOCTOR_ROLE:
		var err error
		doctorID, err = h.dataAPI.GetDoctorIdFromAccountId(ctx.AccountId)
		if err != nil {
			WriteError(err, w, r)
			return
		}
	case api.ADMIN_ROLE:
		// The doctor_id will be parsed in the get/put handlers
	default:
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case HTTP_GET:
		h.get(w, r, doctorID, ctx)
	case HTTP_PUT:
		h.put(w, r, doctorID, ctx)
	default:
		http.NotFound(w, r)
	}
}

func (h *doctorSavedMessageHandler) get(w http.ResponseWriter, r *http.Request, doctorID int64, ctx *Context) {
	if doctorID == 0 {
		// Admin access
		var err error
		doctorID, err = strconv.ParseInt(r.FormValue("doctor_id"), 10, 64)
		if err != nil {
			WriteUserError(w, http.StatusBadRequest, "doctor_id is required")
			return
		}
	}
	msg, err := h.dataAPI.GetSavedMessageForDoctor(doctorID)
	if err == api.NoRowsError {
		msg = ""
	} else if err != nil {
		WriteError(err, w, r)
		return
	}
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &doctorSavedMessageGetResponse{Message: msg})
}

func (h *doctorSavedMessageHandler) put(w http.ResponseWriter, r *http.Request, doctorID int64, ctx *Context) {
	var req doctorSavedMessagePutRequest
	if err := DecodeRequestData(&req, r); err != nil {
		WriteValidationError(err.Error(), w, r)
		return
	}
	if doctorID == 0 {
		// Admin access
		doctorID = req.DoctorID
		if doctorID == 0 {
			WriteValidationError("doctor_id is required", w, r)
			return
		}
	}
	if err := h.dataAPI.SetSavedMessageForDoctor(doctorID, req.Message); err != nil {
		WriteError(err, w, r)
		return
	}
	WriteJSONSuccess(w)
}
