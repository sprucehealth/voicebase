package handlers

import (
	"fmt"
	"net/http"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/mediaproxy"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/svc/products"
)

const productScrapeHTTPCacheDuration = 7 * 24 * time.Hour

type productsScrapeHandler struct {
	svc       products.Service
	proxyRoot string
	proxySvc  *mediaproxy.Service
}

// NewProductsScrape returns a new product scrape handler.
func NewProductsScrape(svc products.Service, proxyRoot string, proxySvc *mediaproxy.Service) httputil.ContextHandler {
	return httputil.SupportedMethods(&productsScrapeHandler{
		svc:       svc,
		proxyRoot: proxyRoot,
		proxySvc:  proxySvc,
	}, httputil.Get)
}

func (h *productsScrapeHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	earl := r.FormValue("url")

	if httputil.CheckAndSetETag(w, r, httputil.GenETag(time.Now().Format("2006-01-02")+":"+earl)) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	p, err := h.svc.Scrape(earl)
	if err != nil {
		if _, ok := errors.Cause(err).(products.ErrScrapeFailed); ok {
			apiservice.WriteUserError(w, http.StatusBadRequest, fmt.Sprintf("Invalid URL"))
			return
		}
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.CacheHeaders(w.Header(), time.Time{}, productScrapeHTTPCacheDuration)
	httputil.JSONResponse(w, http.StatusOK, responses.ProductGETResponse{
		Product: &responses.Product{
			ID:         p.ID,
			Name:       p.Name,
			ImageURLs:  mapMediaProxyURLs(h.proxyRoot, h.proxySvc, p.ImageURLs),
			ProductURL: p.ProductURL,
		},
	})
}
