package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type doctorAPIHandler struct {
	dataAPI api.DataAPI
}

func NewDoctorAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&doctorAPIHandler{
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *doctorAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	doctor, err := h.dataAPI.GetDoctorFromId(doctorID)
	if err == api.NoRowsError {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	www.JSONResponse(w, r, http.StatusOK, &struct {
		Doctor *common.Doctor `json:"doctor"`
	}{
		Doctor: doctor,
	})
}
