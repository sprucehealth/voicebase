package media

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
)

type handler struct {
	dataAPI            api.DataAPI
	store              storage.Store
	expirationDuration time.Duration
}

type uploadResponse struct {
	MediaID  int64  `json:"media_id,string"`
	PhotoID  int64  `json:"photo_id,string"`
	MediaURL string `json:"media_url"`
	PhotoURL string `json:"photo_url"`
}

type mediaResponse struct {
	MediaType string `json:"media_type"`
	MediaURL  string `json:"media_url"`
}

func NewHandler(dataAPI api.DataAPI, store storage.Store, expirationDuration time.Duration) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&handler{
				dataAPI:            dataAPI,
				store:              store,
				expirationDuration: expirationDuration,
			}), []string{"POST"})
}

func (h *handler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	role := ctxt.Role
	var personID int64
	switch role {
	case api.DOCTOR_ROLE, api.MA_ROLE:
		doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
		if err != nil {
			return false, err
		}
		personID, err = h.dataAPI.GetPersonIDByRole(role, doctorID)
		if err != nil {
			return false, err
		}

	case api.PATIENT_ROLE:
		patientID, err := h.dataAPI.GetPatientIDFromAccountID(ctxt.AccountID)
		if err != nil {
			return false, err
		}
		personID, err = h.dataAPI.GetPersonIDByRole(api.PATIENT_ROLE, patientID)
		if err != nil {
			return false, err
		}

	default:
		return false, apiservice.NewAccessForbiddenError()
	}
	ctxt.RequestCache[apiservice.PersonID] = personID
	return true, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	personID := ctxt.RequestCache[apiservice.PersonID].(int64)
	file, handler, err := r.FormFile("media")
	if err != nil {
		file, handler, err = r.FormFile("photo")
		if err != nil {
			apiservice.WriteUserError(w, http.StatusBadRequest, "Missing or invalid media in parameters: "+err.Error())
			return
		}
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
		return
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

	signedURL, err := h.store.GetSignedURL(url, time.Now().Add(h.expirationDuration))
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	res := &uploadResponse{
		MediaID:  id,
		PhotoID:  id,
		MediaURL: signedURL,
		PhotoURL: signedURL,
	}
	apiservice.WriteJSON(w, res)
}
