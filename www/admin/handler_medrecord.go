package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/medrecord"
)

type medicalRecordHandler struct {
	dataAPI api.DataAPI
	r       *medrecord.Renderer
}

func newMedicalRecordHandler(
	dataAPI api.DataAPI,
	diagnosisSvc diagnosis.API,
	mediaStore *media.Store,
	apiDomain string,
	webDomain string,
	signer *sig.Signer,
) httputil.ContextHandler {
	return httputil.SupportedMethods(
		&medicalRecordHandler{
			dataAPI: dataAPI,
			r: &medrecord.Renderer{
				DataAPI:            dataAPI,
				DiagnosisSvc:       diagnosisSvc,
				MediaStore:         mediaStore,
				APIDomain:          apiDomain,
				WebDomain:          webDomain,
				Signer:             signer,
				ExpirationDuration: time.Hour,
			},
		}, httputil.Get)
}

func (h *medicalRecordHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	patientID, err := strconv.ParseInt(r.FormValue("patient_id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	patient, err := h.dataAPI.Patient(patientID, false)
	if api.IsErrNotFound(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	html, err := h.r.Render(patient, medrecord.ROIncludeUnsubmittedVisits)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}
