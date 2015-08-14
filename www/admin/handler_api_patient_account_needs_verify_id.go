package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
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

func newPatientAccountNeedsVerifyIDHandler(needsIDMarker needsIDMarker) httputil.ContextHandler {
	return httputil.SupportedMethods(&patientAccountNeedsVerifyIDHandler{needsIDMarker: needsIDMarker}, httputil.Post)
}

func (h *patientAccountNeedsVerifyIDHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, err := common.ParsePatientID(mux.Vars(ctx)["id"])
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	switch r.Method {
	case httputil.Post:
		rd, err := h.parsePOSTRequest(ctx, r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePOST(w, r, rd, id)
	}
}

func (h *patientAccountNeedsVerifyIDHandler) parsePOSTRequest(ctx context.Context, r *http.Request) (*patientAccountNeedsVerifyIDPOSTRequest, error) {
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
