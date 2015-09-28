package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/svc/products"
	"golang.org/x/net/context"
)

type productsHandler struct {
	svc products.Service
}

// NewProducts returns a new product search handler.
func NewProducts(svc products.Service) httputil.ContextHandler {
	return httputil.SupportedMethods(&productsHandler{
		svc: svc,
	}, httputil.Get)
}

func (h *productsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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
			ID:        p.ID,
			Name:      p.Name,
			ImageURLs: p.ImageURLs,
		}
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}
