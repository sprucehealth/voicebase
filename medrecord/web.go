package medrecord

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/environment"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/www"
)

type downloadHandler struct {
	dataAPI api.DataAPI
	store   storage.Store
}

type photoHandler struct {
	dataAPI    api.DataAPI
	mediaStore *media.Store
	signer     *sig.Signer
}

func NewWebDownloadHandler(dataAPI api.DataAPI, store storage.Store) http.Handler {
	if store == nil {
		log.Fatalf("Medical record handler storage is nil")
	}
	return httputil.SupportedMethods(&downloadHandler{
		dataAPI: dataAPI,
		store:   store,
	}, httputil.Get)
}

func NewPhotoHandler(dataAPI api.DataAPI, mediaStore *media.Store, signer *sig.Signer) http.Handler {
	if mediaStore == nil {
		log.Fatalf("Medical record photo handler storage is nil")
	}
	return httputil.SupportedMethods(&photoHandler{
		dataAPI:    dataAPI,
		mediaStore: mediaStore,
		signer:     signer,
	}, httputil.Get)
}

func (h *downloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	patientID, err := h.dataAPI.GetPatientIDFromAccountID(account.ID)
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
	mediaID, err := strconv.ParseInt(mux.Vars(r)["media"], 10, 64)
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

	// Always validate signature in prod. In other environments
	// alow admins to view the images.
	if environment.IsProd() || (account.Role != api.RoleAdmin) {
		patientID, err := h.dataAPI.GetPatientIDFromAccountID(account.ID)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		if !h.signer.Verify([]byte(fmt.Sprintf("patient:%d:media:%d", patientID, mediaID)), sig) {
			http.NotFound(w, r)
			return
		}
	}

	media, err := h.dataAPI.GetMedia(mediaID)
	if api.IsErrNotFound(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	rc, head, err := h.mediaStore.GetReader(media.URL)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", media.Mimetype)
	w.Header().Set("Content-Length", head.Get("Content-Length"))
	io.Copy(w, rc)
}
