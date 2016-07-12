package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
)

type visitSKUListHandler struct {
	dataAPI api.DataAPI
}

type visitSKUListResponse struct {
	SKUs []string `json:"skus"`
}

func newVisitSKUListHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&visitSKUListHandler{
		dataAPI: dataAPI,
	}, httputil.Get)
}

func (h *visitSKUListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
	}
}

func (h *visitSKUListHandler) get(w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(r.Context())
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
