package handlers

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/svc/products"
	"golang.org/x/net/context"
)

type productsScrapeHandler struct {
	svc products.Service
}

// NewProductsScrape returns a new product scrape handler.
func NewProductsScrape(svc products.Service) httputil.ContextHandler {
	return httputil.SupportedMethods(&productsScrapeHandler{
		svc: svc,
	}, httputil.Get)
}

func (h *productsScrapeHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	earl := r.FormValue("url")
	p, err := h.svc.Scrape(earl)
	if err != nil {
		if _, ok := errors.Cause(err).(products.ErrScrapeFailed); ok {
			apiservice.WriteUserError(w, http.StatusBadRequest, fmt.Sprintf("Invalid URL"))
			return
		}
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, responses.ProductGETResponse{
		Product: &responses.Product{
			ID:         p.ID,
			Name:       p.Name,
			ImageURLs:  p.ImageURLs,
			ProductURL: p.ProductURL,
		},
	})
}
