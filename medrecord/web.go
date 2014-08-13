package medrecord

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/www"
)

type downloadHandler struct {
	dataAPI api.DataAPI
	store   storage.Store
}

type photoHandler struct {
	dataAPI api.DataAPI
	store   storage.Store
	signer  *common.Signer
}

func NewWebDownloadHandler(dataAPI api.DataAPI, store storage.Store) http.Handler {
	return httputil.SupportedMethods(&downloadHandler{
		dataAPI: dataAPI,
		store:   store,
	}, []string{"GET"})
}

func NewPhotoHandler(dataAPI api.DataAPI, store storage.Store, signer *common.Signer) http.Handler {
	return httputil.SupportedMethods(&photoHandler{
		dataAPI: dataAPI,
		store:   store,
		signer:  signer,
	}, []string{"GET"})
}

func (h *downloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	patientID, err := h.dataAPI.GetPatientIdFromAccountId(account.ID)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	records, err := h.dataAPI.MedicalRecordsForPatient(patientID)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	var latest *common.MedicalRecord
	for _, mr := range records {
		if mr.Status == common.MRSuccess {
			latest = mr
			break
		}
	}

	if latest == nil {
		http.NotFound(w, r)
		return
	}

	// TODO: once storage supports signed URLs switch to a redirect instead of directly serving the file

	rc, head, err := h.store.GetReader(latest.StorageURL)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", head.Get("Content-Type"))
	w.Header().Set("Content-Length", head.Get("Content-Length"))
	io.Copy(w, rc)
}

func (h *photoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	photoID, err := strconv.ParseInt(mux.Vars(r)["photo"], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	sig, err := base64.URLEncoding.DecodeString(r.FormValue("s"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)
	patientID, err := h.dataAPI.GetPatientIdFromAccountId(account.ID)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	if !h.signer.Verify([]byte(fmt.Sprintf("patient:%d:photo:%d", patientID, photoID)), sig) {
		http.NotFound(w, r)
		return
	}

	photo, err := h.dataAPI.GetPhoto(photoID)
	if err == api.NoRowsError {
		http.NotFound(w, r)
		return
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	rc, head, err := h.store.GetReader(photo.URL)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", photo.Mimetype)
	w.Header().Set("Content-Length", head.Get("Content-Length"))
	io.Copy(w, rc)
}
