package admin

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"
)

type globalFTPHandler struct {
	dataAPI    api.DataAPI
	mediaStore *media.Store
}

type globalFTPGETResponse struct {
	FavoriteTreatmentPlans []*responses.FavoriteTreatmentPlan `json:"favorite_treatment_plans"`
}

type globalFTPGETRequest struct {
	Lifecycles []string `schema:"lifecycles"`
}

func newGlobalFTPHandler(
	dataAPI api.DataAPI,
	mediaStore *media.Store) httputil.ContextHandler {
	return httputil.SupportedMethods(&globalFTPHandler{dataAPI: dataAPI, mediaStore: mediaStore}, httputil.Get)
}

func (h *globalFTPHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		request, err := h.parseGETRequest(ctx, r)
		if err != nil {
			www.APIBadRequestError(w, r, "Unable to parse request")
			return
		}
		h.serveGET(w, r, request)
	}
}

func (h *globalFTPHandler) parseGETRequest(ctx context.Context, r *http.Request) (*globalFTPGETRequest, error) {
	rd := &globalFTPGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if len(rd.Lifecycles) == 0 {
		rd.Lifecycles = append(rd.Lifecycles, "ACTIVE")
	}

	return rd, nil
}

func (h *globalFTPHandler) serveGET(w http.ResponseWriter, r *http.Request, rd *globalFTPGETRequest) {
	gftps, err := h.dataAPI.GlobalFavoriteTreatmentPlans(rd.Lifecycles)
	if err != nil {
		golog.Errorf("Unable to lookup Global FTPs with Lifecycles %v", rd.Lifecycles)
		www.APIInternalError(w, r, err)
		return
	}

	response := globalFTPGETResponse{
		FavoriteTreatmentPlans: make([]*responses.FavoriteTreatmentPlan, len(gftps)),
	}
	for i, ftp := range gftps {
		ftpr, err := responses.TransformFTPToResponse(h.dataAPI, h.mediaStore, scheduledMessageMediaExpirationDuration, ftp, "")
		if err != nil {
			golog.Errorf("Unable to lookup transform FTP into response.")
			www.APIInternalError(w, r, err)
			return
		}

		response.FavoriteTreatmentPlans[i] = ftpr
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}
