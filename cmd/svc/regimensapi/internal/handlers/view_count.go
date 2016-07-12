package handlers

import (
	"net/http"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
)

type viewIncrementer interface {
	IncrementViewCount(id string) error
}

type viewCountHandler struct {
	svc viewIncrementer
}

// NewViewCount returns an initialized instance of viewCountHandler
func NewViewCount(svc viewIncrementer) httputil.ContextHandler {
	return httputil.SupportedMethods(&viewCountHandler{svc: svc}, httputil.Post)
}

func (h *viewCountHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, ok := mux.Vars(ctx)["id"]
	if !ok {
		apiservice.WriteResourceNotFoundError(ctx, "an id must be provided", w, r)
		return
	}

	switch r.Method {
	case httputil.Post:
		h.servePOST(ctx, w, r, id)
	}
}

func (h *viewCountHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, id string) {
	conc.Go(func() { h.svc.IncrementViewCount(id) })
	apiservice.WriteJSONSuccess(w)
}
