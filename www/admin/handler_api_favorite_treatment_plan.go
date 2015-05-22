package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/responses"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type ftpHandler struct {
	dataAPI    api.DataAPI
	mediaStore *media.Store
}

type ftpGETResponse struct {
	FavoriteTreatmentPlan *responses.FavoriteTreatmentPlan `json:"favorite_treatment_plan"`
}

func NewFTPHandler(dataAPI api.DataAPI, mediaStore *media.Store) http.Handler {
	return httputil.SupportedMethods(&ftpHandler{dataAPI: dataAPI, mediaStore: mediaStore}, httputil.Get)
}

func (h *ftpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ftpID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	switch r.Method {
	case "GET":
		h.serveGET(w, r, ftpID)
	}
}

func (h *ftpHandler) serveGET(w http.ResponseWriter, r *http.Request, ftpID int64) {
	ftp, err := h.dataAPI.FavoriteTreatmentPlan(ftpID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	ftpr, err := responses.TransformFTPToResponse(h.dataAPI, h.mediaStore, scheduledMessageMediaExpirationDuration, ftp, "")
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, ftpGETResponse{
		FavoriteTreatmentPlan: ftpr,
	})
}
