package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

// TODO mraines: This should be an intermidiate service latyer rather than a direct DAL
type practiceModelDAL interface {
	PracticeModel(doctorID int64) (*common.PracticeModel, error)
	UpdatePracticeModel(doctorID int64, pmu *common.PracticeModelUpdate) (int64, error)
}

type practiceModelHandler struct {
	practiceModelDAL practiceModelDAL
}

// PracticeModelGETResponse represents the data expected to returned from a successful GET request
type PracticeModelGETResponse struct {
	PracticeModel *responses.PracticeModel `json:"practice_model"`
}

// PracticeModelPUTRequest represents the data expected to be associated with a successful POST request
type PracticeModelPUTRequest struct {
	IsSprucePC           *bool `json:"is_spruce_pc"`
	HasPracticeExtension *bool `json:"has_practice_extension"`
}

// newPracticeModelHandler returns an initialized instance of practiceModelHandler
func newPracticeModelHandler(practiceModelDAL practiceModelDAL) httputil.ContextHandler {
	return httputil.SupportedMethods(&practiceModelHandler{practiceModelDAL: practiceModelDAL}, httputil.Get, httputil.Put)
}

func (h *practiceModelHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	providerID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	switch r.Method {
	case httputil.Get:
		h.serveGET(ctx, w, r, providerID)
	case httputil.Put:
		rd, err := h.parsePUTRequest(ctx, r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePUT(ctx, w, r, rd, providerID)
	}
}

func (h *practiceModelHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, doctorID int64) {
	pm, err := h.practiceModelDAL.PracticeModel(doctorID)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &PracticeModelGETResponse{PracticeModel: responses.TransformPracticeModel(pm)})
}

func (h *practiceModelHandler) parsePUTRequest(ctx context.Context, r *http.Request) (*PracticeModelPUTRequest, error) {
	rd := &PracticeModelPUTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *practiceModelHandler) servePUT(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *PracticeModelPUTRequest, doctorID int64) {
	if _, err := h.practiceModelDAL.UpdatePracticeModel(doctorID, &common.PracticeModelUpdate{
		IsSprucePC:           rd.IsSprucePC,
		HasPracticeExtension: rd.HasPracticeExtension,
	}); err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	h.serveGET(ctx, w, r, doctorID)
}
