package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type visitSKUListHandler struct {
	dataAPI api.DataAPI
}

type visitSKUListResponse struct {
	SKUs []string `json:"skus"`
}

func NewVisitSKUListHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&visitSKUListHandler{
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *visitSKUListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		h.get(w, r)
	}
}

func (h *visitSKUListHandler) get(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "GetVisitSKUList", nil)

	var activeOnly bool
	if s := r.FormValue("active_only"); s != "" {
		var err error
		activeOnly, err = strconv.ParseBool(s)
		if err != nil {
			www.APIBadRequestError(w, r, "failed to parse active_only")
			return
		}
	}

	skus, err := h.dataAPI.VisitSKUs(activeOnly)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &visitSKUListResponse{SKUs: skus})
}
