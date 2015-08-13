package home

import (
	"io"
	"log"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/www"
)

type medRecordDownloadHandler struct {
	dataAPI api.DataAPI
	store   storage.Store
}

func newMedRecordWebDownloadHandler(dataAPI api.DataAPI, store storage.Store) httputil.ContextHandler {
	if store == nil {
		log.Fatalf("Medical record handler storage is nil")
	}
	return httputil.SupportedMethods(
		www.RoleRequiredHandler(
			&medRecordDownloadHandler{
				dataAPI: dataAPI,
				store:   store,
			}, nil, api.RolePatient), httputil.Get)
}

func (h *medRecordDownloadHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
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
