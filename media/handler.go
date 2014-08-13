package media

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

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

func (h *handler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	role := apiservice.GetContext(r).Role
	var personId int64
	switch role {
	default:
		return false, apiservice.NewAccessForbiddenError()
	case api.DOCTOR_ROLE, api.MA_ROLE:
		doctorId, err := h.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
		if err == nil {
			personId, err = h.dataAPI.GetPersonIdByRole(api.DOCTOR_ROLE, doctorId)
		}
		if err != nil {
			return false, apiservice.NewResourceNotFoundError("Failed to get person object for doctor:"+err.Error(), r)
		}
	case api.PATIENT_ROLE:
		patientId, err := h.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
		if err == nil {
			personId, err = h.dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, patientId)
		}
		if err != nil {
			return false, apiservice.NewResourceNotFoundError("Failed to get person object for patient:"+err.Error(), r)
		}
	}
	ctxt.RequestCache[apiservice.PersonID] = personId
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

func (h *handler) upload(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	personID := ctxt.RequestCache[apiservice.PersonID].(int64)
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

	id, err := h.dataAPI.AddMedia(personID, url, contentType)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &uploadResponse{
		MediaID: id,
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}
