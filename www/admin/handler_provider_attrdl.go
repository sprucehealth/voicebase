package admin

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/www"
)

type providerAttrDownloadHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
	store   storage.Store
}

func NewProviderAttrDownloadHandler(router *mux.Router, dataAPI api.DataAPI, store storage.Store) http.Handler {
	return httputil.SupportedMethods(&providerAttrDownloadHandler{
		router:  router,
		dataAPI: dataAPI,
		store:   store,
	}, httputil.Get)
}

func (h *providerAttrDownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	providerID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	attrName := vars["attr"]

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "Admin", "DownloadProviderAttributeFile", map[string]interface{}{"provider_id": providerID, "attribute": attrName})

	attr, err := h.dataAPI.DoctorAttributes(providerID, []string{attrName})
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	if len(attr) == 0 {
		http.NotFound(w, r)
		return
	}

	rc, headers, err := h.store.GetReader(attr[attrName])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()

	hd := w.Header()
	hd.Set("Content-Type", headers.Get("Content-Type"))
	hd.Set("Content-Length", headers.Get("Content-Length"))
	if fn := headers.Get("X-Amz-Meta-Original-Name"); fn != "" {
		hd.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fn))
	}
	io.Copy(w, rc)
}