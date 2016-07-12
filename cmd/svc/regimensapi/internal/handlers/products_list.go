package handlers

import (
	"net/http"
	"sync"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/mediaproxy"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/products"
)

const productListHTTPCacheDuration = 24 * time.Hour

type productsListHandler struct {
	svc       products.Service
	proxyRoot string
	proxySvc  *mediaproxy.Service
}

// NewProductsList returns a new product search handler.
func NewProductsList(svc products.Service, proxyRoot string, proxySvc *mediaproxy.Service) httputil.ContextHandler {
	return httputil.SupportedMethods(&productsListHandler{
		svc:       svc,
		proxyRoot: proxyRoot,
		proxySvc:  proxySvc,
	}, httputil.Get)
}

func (h *productsListHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")

	if httputil.CheckAndSetETag(w, r, httputil.GenETag(time.Now().Format("2006-01-02")+":"+query)) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	prods, err := h.svc.Search(query)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	res := &responses.ProductList{
		Products: make([]*responses.Product, len(prods)),
	}
	var wg sync.WaitGroup
	wg.Add(len(prods))
	for i, p := range prods {
		go func(i int, p *products.Product) {
			defer wg.Done()
			res.Products[i] = &responses.Product{
				ID:         p.ID,
				Name:       p.Name,
				ImageURLs:  mapMediaProxyURLs(h.proxyRoot, h.proxySvc, p.ImageURLs),
				ProductURL: p.ProductURL,
				Prefetched: ptr.Bool(true),
			}
		}(i, p)
	}
	wg.Wait()

	httputil.CacheHeaders(w.Header(), time.Time{}, productListHTTPCacheDuration)
	httputil.JSONResponse(w, http.StatusOK, res)
}
