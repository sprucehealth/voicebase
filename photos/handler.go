package photos

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
)

type Handler struct {
	dataAPI api.DataAPI
	store   storage.Store
}

type getRequest struct {
	PhotoId     int64  `schema:"photo_id,required"`
	ClaimerType string `schema:"claimer_type,required"`
	ClaimerId   int64  `schema:"claimer_id,required"`
}

type uploadResponse struct {
	PhotoId int64 `json:"photo_id,string"`
}

func NewHandler(dataAPI api.DataAPI, store storage.Store) *Handler {
	return &Handler{
		dataAPI: dataAPI,
		store:   store,
	}
}

func (h *Handler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		h.get(w, r)
	case apiservice.HTTP_POST:
		h.upload(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var req getRequest
	if err := schema.NewDecoder().Decode(&req, r.Form); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	photo, err := h.dataAPI.GetPhoto(req.PhotoId)
	if err == api.NoRowsError {
		http.NotFound(w, r)
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get photo: "+err.Error())
		return
	}

	// TODO: need a more robust check for verifying access rights
	if photo.ClaimerType != req.ClaimerType || photo.ClaimerId != req.ClaimerId {
		http.NotFound(w, r)
		return
	}

	rc, header, err := h.store.GetReader(photo.URL)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get photo: "+err.Error())
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", header.Get("Content-Type"))
	w.Header().Set("Content-Length", header.Get("Content-Length"))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		golog.Errorf("Failed to send photo image: %s", err.Error())
	}
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	var personId int64
	doctorId, err := h.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err == nil {
		personId, err = h.dataAPI.GetPersonIdByRole(api.DOCTOR_ROLE, doctorId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "failed to get person object for doctor: "+err.Error())
			return
		}
	} else if patientId, err := h.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId); err == nil {
		personId, err = h.dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, patientId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "failed to get person object for patient: "+err.Error())
			return
		}
	} else {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "failed to get patient or doctor: "+err.Error())
		return
	}

	file, handler, err := r.FormFile("photo")
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Missing or invalid photo in parameters: "+err.Error())
		return
	}
	defer file.Close()

	size, err := common.SeekerSize(file)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	uid := make([]byte, 16)
	if _, err := rand.Read(uid); err != nil {
		apiservice.WriteError(err, w, r)
	}
	name := "photo-" + hex.EncodeToString(uid)
	contentType := handler.Header.Get("Content-Type")
	headers := http.Header{
		"Content-Type": []string{contentType},
	}

	url, err := h.store.PutReader(name, file, size, headers)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	id, err := h.dataAPI.AddPhoto(personId, url, contentType)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &uploadResponse{
		PhotoId: id,
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}
