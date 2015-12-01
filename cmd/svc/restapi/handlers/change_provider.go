package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type changeProviderService interface {
	ChangeCareProvider(caseID int64, desiredProviderID int64, changeAuthorDoctorID int64) error
	ElligibleCareProvidersForCase(caseID int64) ([]*common.Doctor, error)
}

type changeProviderHandler struct {
	svc changeProviderService
}

type changeProviderGETRequest struct {
	CaseID int64 `schema:"case_id,required"`
}

type changeProviderGETResponse struct {
	ElligibleDoctors []*common.Doctor `json:"elligible_doctors"`
}

type changeProviderPOSTRequest struct {
	CaseID   int64 `json:"case_id,string"`
	DoctorID int64 `json:"doctor_id,string"`
}

// NewChangeProviderHandler returns an initialized instance of changeProviderHandler
func NewChangeProviderHandler(svc changeProviderService) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			&changeProviderHandler{svc: svc}, api.RoleCC), httputil.Post, httputil.Get)
}

func (h *changeProviderHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		rd, err := h.parseGETRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		doctors, err := h.svc.ElligibleCareProvidersForCase(rd.CaseID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
		httputil.JSONResponse(w, http.StatusOK, &changeProviderGETResponse{ElligibleDoctors: doctors})
	case httputil.Post:
		rd, err := h.parsePOSTRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}

		caller, ok := apiservice.CtxCC(ctx)
		if !ok {
			apiservice.WriteBadRequestError(ctx, errors.New("No care coordinator found in context"), w, r)
			return
		}
		if err := h.svc.ChangeCareProvider(rd.CaseID, rd.DoctorID, caller.ID.Int64()); err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
		apiservice.WriteJSONSuccess(w)
	}
}

func (h *changeProviderHandler) parseGETRequest(ctx context.Context, r *http.Request) (*changeProviderGETRequest, error) {
	rd := &changeProviderGETRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return nil, err
	}
	return rd, nil
}

func (h *changeProviderHandler) parsePOSTRequest(ctx context.Context, r *http.Request) (*changeProviderPOSTRequest, error) {
	rd := &changeProviderPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.DoctorID == 0 || rd.CaseID == 0 {
		return nil, fmt.Errorf("doctor_id, case_id required")
	}
	return rd, nil
}
