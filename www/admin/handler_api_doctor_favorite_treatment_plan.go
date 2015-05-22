package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"
)

const scheduledMessageMediaExpirationDuration = time.Hour * 24 * 7

type doctorFTPHandler struct {
	dataAPI    api.DataAPI
	mediaStore *media.Store
}

type doctorFTPGETResponse struct {
	FavoriteTreatmentPlans map[string][]*responses.FavoriteTreatmentPlan `json:"favorite_treatment_plans"`
}

func NewDoctorFTPHandler(
	dataAPI api.DataAPI,
	mediaStore *media.Store) http.Handler {
	return httputil.SupportedMethods(&doctorFTPHandler{dataAPI: dataAPI, mediaStore: mediaStore}, httputil.Get)
}

func (h *doctorFTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	switch r.Method {
	case "GET":
		h.serveGET(w, r, doctorID)
	}
}

func (h *doctorFTPHandler) serveGET(w http.ResponseWriter, r *http.Request, doctorID int64) {
	memberships, err := h.dataAPI.FTPMembershipsForDoctor(doctorID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	response := doctorFTPGETResponse{
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

		if _, ok := response.FavoriteTreatmentPlans[pathway.Name]; !ok {
			response.FavoriteTreatmentPlans[pathway.Name] = make([]*responses.FavoriteTreatmentPlan, 0)
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
