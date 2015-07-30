package home

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/common"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
)

type parentalMedicalRecordHandler struct {
	dataAPI api.DataAPI
	r       medicalRecordRenderer
}

type medicalRecordRenderer interface {
	Render(*common.Patient) ([]byte, error)
}

func newParentalMedicalRecordHandler(
	dataAPI api.DataAPI,
	renderer medicalRecordRenderer,
) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(
		www.RoleRequiredHandler(
			&parentalMedicalRecordHandler{
				dataAPI: dataAPI,
				r:       renderer,
			}, nil, api.RolePatient),
		httputil.Get)
}

func (h *parentalMedicalRecordHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	childPatientID, err := strconv.ParseInt(mux.Vars(ctx)["childid"], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	account := www.MustCtxAccount(ctx)
	parentPatientID, err := h.dataAPI.GetPatientIDFromAccountID(account.ID)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	// Make sure the person loading this page (assuming the parent) has a link with the child they're
	// trying to view the medical record for.
	consent, err := h.dataAPI.ParentalConsent(childPatientID)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	var con *common.ParentalConsent
	for _, c := range consent {
		if c.ParentPatientID == parentPatientID {
			con = c
			break
		}
	}
	if con == nil {
		http.NotFound(w, r)
		return
	}

	patient, err := h.dataAPI.Patient(childPatientID, false)
	if api.IsErrNotFound(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	html, err := h.r.Render(patient)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}
