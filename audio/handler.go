package audio

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
	AudioID     int64  `schema:"audio_id, required"`
	ClaimerType string `schema:"claimer_type, required"`
	ClaimerID   int64  `schema:"claimer_id, required"`
}

type uploadResponse struct {
	AudioID int64 `json:"audio_id,string"`
}

func NewHandler(dataAPI api.DataAPI, store storage.Store) *Handler {
	return &Handler{
		dataAPI: dataAPI,
		store:   store,
	}
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
	requestData := &getRequest{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	} else if requestData.AudioID == 0 || requestData.ClaimerID == 0 {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "audio_id and claimer_id ")
		return
	}

	audio, err := h.dataAPI.GetAudio(requestData.AudioID)
	if err == api.NoRowsError {
		http.NotFound(w, r)
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get audio: "+err.Error())
		return
	}

	// TODO: need a more robust check for verifying access rights
	if audio.ClaimerID != requestData.ClaimerID {
		http.NotFound(w, r)
		return
	}

	rc, header, err := h.store.GetReader(audio.URL)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get audio: "+err.Error())
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", header.Get("Content-Type"))
	w.Header().Set("Content-Length", header.Get("Content-Length"))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		golog.Errorf("Failed to send audio: %s", err.Error())
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

	file, handler, err := r.FormFile("audio")
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Missing or invalid audio in parameters: "+err.Error())
		return
	}
	defer file.Close()

	size, err := common.SeekerSize(file)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	golog.Infof("The file size is %d", size)
	uid := make([]byte, 16)
	if _, err := rand.Read(uid); err != nil {
		apiservice.WriteError(err, w, r)
	}
	name := "audio-" + hex.EncodeToString(uid)
	contentType := handler.Header.Get("Content-Type")
	golog.Infof("The content type is %s", contentType)
	headers := http.Header{
		"Content-Type": []string{contentType},
	}

	url, err := h.store.PutReader(name, file, size, headers)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	golog.Infof("the url is %s", url)
	id, err := h.dataAPI.AddAudio(personId, url, contentType)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &uploadResponse{
		AudioID: id,
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}
