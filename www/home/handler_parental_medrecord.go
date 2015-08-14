package home

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/medrecord"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type parentalMedicalRecordHandler struct {
	dataAPI api.DataAPI
	r       medicalRecordRenderer
}

type medicalRecordRenderer interface {
	Render(*common.Patient, medrecord.RenderOption) ([]byte, error)
}

func newParentalMedicalRecordHandler(
	dataAPI api.DataAPI,
	renderer medicalRecordRenderer,
) httputil.ContextHandler {
	return httputil.SupportedMethods(
		www.RoleRequiredHandler(
			&parentalMedicalRecordHandler{
				dataAPI: dataAPI,
				r:       renderer,
			}, nil, api.RolePatient),
		httputil.Get)
}

func (h *parentalMedicalRecordHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	childPatientID, err := common.ParsePatientID(mux.Vars(ctx)["childid"])
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
	// If the patient doesn't have consent yet then redirect back to the flow
	// as it's not yet complete.
	if !patient.HasParentalConsent {
		http.Redirect(w, r, fmt.Sprintf("/pc/%d/start", patient.ID.Int64()), http.StatusSeeOther)
		return
	}
	html, err := h.r.Render(patient, medrecord.ROIncludeUnsubmittedVisits)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}
