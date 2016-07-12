package handlers

import (
	"net/http"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/mediaproxy"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/products"
)

const productHTTPCacheDuration = 24 * time.Hour

type productsHandler struct {
	svc       products.Service
	proxyRoot string
	proxySvc  *mediaproxy.Service
}

// NewProducts returns a new single product handler
func NewProducts(svc products.Service, proxyRoot string, proxySvc *mediaproxy.Service) httputil.ContextHandler {
	return httputil.SupportedMethods(&productsHandler{
		svc:       svc,
		proxyRoot: proxyRoot,
		proxySvc:  proxySvc,
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
			ImageURLs:  mapMediaProxyURLs(h.proxyRoot, h.proxySvc, p.ImageURLs),
			ProductURL: p.ProductURL,
		},
	})
}

func mapMediaProxyURLs(proxyRoot string, proxySvc *mediaproxy.Service, imageURLs []string) []string {
	if proxySvc == nil {
		return imageURLs
	}
	media, err := proxySvc.LookupByURL(imageURLs)
	if err != nil {
		golog.Errorf("Failed to map media proxy URLs: %s", err)
		// Just return the external URLs as it's an OK fallback
		return imageURLs
	}
	iu := make([]string, 0, len(media))
	for _, m := range media {
		if m.Status != mediaproxy.StatusFailedPerm {
			iu = append(iu, proxyRoot+m.ID)
		}
	}
	return iu
}
