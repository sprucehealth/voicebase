package handlers

import (
	"net/http"
	"time"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/products"
	"golang.org/x/net/context"
)

const productHTTPCacheDuration = 24 * time.Hour

type productsHandler struct {
	svc products.Service
}

// NewProducts returns a new single product handler
func NewProducts(svc products.Service) httputil.ContextHandler {
	return httputil.SupportedMethods(&productsHandler{
		svc: svc,
	}, httputil.Get)
}

func (h *productsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	productID := mux.Vars(ctx)["id"]

	if httputil.CheckAndSetETag(w, r, httputil.GenETag(time.Now().Format("2006-01-02")+":"+productID)) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	p, err := h.svc.Lookup(productID)
	if err == products.ErrNotFound {
		apiservice.WriteResourceNotFoundError(ctx, "Product not found", w, r)
		return
	}

	httputil.CacheHeaders(w.Header(), time.Time{}, productHTTPCacheDuration)
	httputil.JSONResponse(w, http.StatusOK, responses.ProductGETResponse{
		Product: &responses.Product{
			ID:         p.ID,
			Name:       p.Name,
			ImageURLs:  p.ImageURLs,
			ProductURL: p.ProductURL,
		},
	})
}
