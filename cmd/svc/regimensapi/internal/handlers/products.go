package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/products"
	"golang.org/x/net/context"
)

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

	p, err := h.svc.Lookup(productID)
	if err == products.ErrNotFound {
		apiservice.WriteResourceNotFoundError(ctx, "Product not found", w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Product *product
	}{
		Product: &product{
			ID:         p.ID,
			Name:       p.Name,
			ImageURLs:  p.ImageURLs,
			ProductURL: p.ProductURL,
		},
	})
}
