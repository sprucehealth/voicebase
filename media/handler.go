package media

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/storage"
)

type handler struct {
	dataAPI api.DataAPI
	store   storage.Store
}

type uploadResponse struct {
	MediaID int64 `json:"media_id,string"`
}

type mediaResponse struct {
	MediaType string `json:"media_type"`
	MediaURL  string `json:"media_url"`
}

func NewHandler(dataAPI api.DataAPI, store storage.Store) *handler {
	return &handler{dataAPI: dataAPI, store: store}
}

func (h *Handler) IsAuthorized(r *http.Request) (bool, error) {
	role := apiservice.GetContext(r).Role
	var personId int64
	switch role {
	default:
		return false, apiservice.NewAccessForbiddenError()
	case api.DOCTOR_ROLE, api.MA_ROLE:
		if doctorId, err := h.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId); err == nil {
			personId, err = h.dataAPI.GetPersonIdByRole(api.DOCTOR_ROLE, doctorId)
		}
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "failed to get person object for doctor: "+err.Error())
			return false, err
		}
	case api.PATIENT_ROLE:
		if patientId, err := h.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId); err == nil {
			personId, err = h.dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, patientId)
		}
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "failed to get person object for patient: "+err.Error())
			return false, err
		}
	}
	ctxt.RequestCache[personId] = personId
	return true, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_POST:
		h.upload(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	personId := ctxt.RequestCache[personId].(int64)
	file, handler, err := r.FormFile("media")
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Missing or invalid media in parameters: "+err.Error())
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

	name := "media-" + hex.EncodeToString(uid)
	contentType := handler.Header.Get("Content-Type")
	headers := http.Header{
		"Content-Type": []string{contentType},
	}

	url, err := h.store.PutReader(name, file, size, headers)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	id, err := h.dataAPI.AddMedia(personId, url, contentType)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &uploadResponse{
		MediaID: id,
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}
