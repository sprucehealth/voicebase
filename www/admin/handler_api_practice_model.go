package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

// TODO mraines: This should be an intermidiate service latyer rather than a direct DAL
type practiceModelDAL interface {
	InitializePracticeModelInAllStates(doctorID int64) error
	PracticeModels(doctorID int64) (map[string]*common.PracticeModel, error)
	UpdatePracticeModel(doctorID, stateID int64, pmu *common.PracticeModelUpdate) (int64, error)
	UpsertPracticeModelInAllStates(doctorID int64, aspmu *common.AllStatesPracticeModelUpdate) (int64, error)
}

type practiceModelHandler struct {
	practiceModelDAL practiceModelDAL
}

// TODO: Both the PUT method here should take ID's rather than abbreviations. This relys on the client building out it's state list from an API call.

// practiceModelGETResponse represents the data expected to returned from a successful GET request
type practiceModelGETResponse struct {
	PracticeModels map[string]*responses.PracticeModel `json:"practice_models"`
}

// practiceModelPUTRequest represents the data expected to be associated with a successful POST request
type practiceModelPUTRequest struct {
	AllStates            bool  `json:"all_states"`
	StateID              int64 `json:"state_id,string"`
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
	// Bootstrap any missing records
	if err := h.practiceModelDAL.InitializePracticeModelInAllStates(doctorID); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	pms, err := h.practiceModelDAL.PracticeModels(doctorID)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	respPMs := make(map[string]*responses.PracticeModel, len(pms))
	for sa, pm := range pms {
		respPMs[sa] = responses.TransformPracticeModel(pm)
	}
	httputil.JSONResponse(w, http.StatusOK, &practiceModelGETResponse{PracticeModels: respPMs})
}

func (h *practiceModelHandler) parsePUTRequest(ctx context.Context, r *http.Request) (*practiceModelPUTRequest, error) {
	rd := &practiceModelPUTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if rd.StateID == 0 && !rd.AllStates {
		return nil, errors.New("state_abbreviation OR all_states required")
	}
	return rd, nil
}

func (h *practiceModelHandler) servePUT(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *practiceModelPUTRequest, doctorID int64) {
	if !rd.AllStates {
		if _, err := h.practiceModelDAL.UpdatePracticeModel(doctorID, rd.StateID, &common.PracticeModelUpdate{
			IsSprucePC:           rd.IsSprucePC,
			HasPracticeExtension: rd.HasPracticeExtension,
		}); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	} else {
		if _, err := h.practiceModelDAL.UpsertPracticeModelInAllStates(doctorID, &common.AllStatesPracticeModelUpdate{
			HasPracticeExtension: rd.HasPracticeExtension,
		}); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	}
	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}
