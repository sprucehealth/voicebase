package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

const scheduledMessageMediaExpirationDuration = time.Hour * 24 * 7

type providerFTPHandler struct {
	dataAPI    api.DataAPI
	mediaStore *media.Store
}

type providerFTPGETResponse struct {
	FavoriteTreatmentPlans map[string][]*responses.FavoriteTreatmentPlan `json:"favorite_treatment_plans"`
}

func newProviderFTPHandler(dataAPI api.DataAPI, mediaStore *media.Store) httputil.ContextHandler {
	return httputil.SupportedMethods(&providerFTPHandler{dataAPI: dataAPI, mediaStore: mediaStore}, httputil.Get)
}

func (h *providerFTPHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	switch r.Method {
	case httputil.Get:
		h.serveGET(w, r, doctorID)
	}
}

func (h *providerFTPHandler) serveGET(w http.ResponseWriter, r *http.Request, doctorID int64) {
	memberships, err := h.dataAPI.FTPMembershipsForDoctor(doctorID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	response := providerFTPGETResponse{
		FavoriteTreatmentPlans: make(map[string][]*responses.FavoriteTreatmentPlan),
	}
	for _, membership := range memberships {
		ftp, err := h.dataAPI.FavoriteTreatmentPlan(membership.DoctorFavoritePlanID)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		pathway, err := h.dataAPI.Pathway(membership.ClinicalPathwayID, api.PONone)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		ftpr, err := responses.TransformFTPToResponse(h.dataAPI, h.mediaStore, scheduledMessageMediaExpirationDuration, ftp, pathway.Tag)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		response.FavoriteTreatmentPlans[pathway.Name] = append(response.FavoriteTreatmentPlans[pathway.Name], ftpr)
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}
