package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/mux"
)

// NeedsIDMarker describes the interface needed to mark an account as needing verification
type needsIDMarker interface {
	MarkForNeedsIDVerification(patientID common.PatientID, promoCode string) error
}

type patientAccountNeedsVerifyIDHandler struct {
	needsIDMarker needsIDMarker
}

// PatientAccountNeedsVerifyIDPOSTRequest represents the data expected to be associated with a successful POST request
type patientAccountNeedsVerifyIDPOSTRequest struct {
	PromoCode string `json:"promo_code"`
}

func newPatientAccountNeedsVerifyIDHandler(needsIDMarker needsIDMarker) http.Handler {
	return httputil.SupportedMethods(&patientAccountNeedsVerifyIDHandler{needsIDMarker: needsIDMarker}, httputil.Post)
}

func (h *patientAccountNeedsVerifyIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := common.ParsePatientID(mux.Vars(r.Context())["id"])
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	switch r.Method {
	case httputil.Post:
		rd, err := h.parsePOSTRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePOST(w, r, rd, id)
	}
}

func (h *patientAccountNeedsVerifyIDHandler) parsePOSTRequest(r *http.Request) (*patientAccountNeedsVerifyIDPOSTRequest, error) {
	rd := &patientAccountNeedsVerifyIDPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if rd.PromoCode == "" {
		return nil, errors.New("promo_code required")
	}
	return rd, nil
}

func (h *patientAccountNeedsVerifyIDHandler) servePOST(w http.ResponseWriter, r *http.Request, rd *patientAccountNeedsVerifyIDPOSTRequest, id common.PatientID) {
	if err := h.needsIDMarker.MarkForNeedsIDVerification(id, rd.PromoCode); api.IsErrNotFound(err) {
		www.APIBadRequestError(w, r, err.Error())
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}
