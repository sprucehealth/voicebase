package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/products"
	"golang.org/x/net/context"
)

type productsListHandler struct {
	svc products.Service
}

// NewProductsList returns a new product search handler.
func NewProductsList(svc products.Service) httputil.ContextHandler {
	return httputil.SupportedMethods(&productsListHandler{
		svc: svc,
	}, httputil.Get)
}

func (h *productsListHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")
	prods, err := h.svc.Search(query)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	res := &productList{
		Products: make([]*product, len(prods)),
	}
	for i, p := range prods {
		res.Products[i] = &product{
			ID:         p.ID,
			Name:       p.Name,
			ImageURLs:  p.ImageURLs,
			ProductURL: p.ProductURL,
			Prefetched: ptr.Bool(true),
		}
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}
