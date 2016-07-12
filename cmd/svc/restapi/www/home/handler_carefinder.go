package home

import (
	"net/http"
	nethttputil "net/http/httputil"
	"net/url"
	"sync/atomic"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/httputil"
)

var carefinderURLDef = &cfg.ValueDef{
	Name:        "CareFinder.URL",
	Description: "URL to which to route requests for carefinder",
	Type:        cfg.ValueTypeString,
	Default:     "",
}

type careFinderHandler struct {
	cfg                  cfg.Store
	currentCareFinderURL string
	reverseProxy         atomic.Value
}

type reverseProxyInfo struct {
	reverseProxy *nethttputil.ReverseProxy
	currentURL   string
}

// NewCareFinderHandler returns a handler that reverse proxies to the carefinder
// service if configured else redirects to the sprucehealth.com main website.
func NewCareFinderHandler(cfg cfg.Store) httputil.ContextHandler {
	cfg.Register(carefinderURLDef)

	return &careFinderHandler{
		cfg: cfg,
	}
}

func (c *careFinderHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	carefinderURL := c.cfg.Snapshot().String(carefinderURLDef.Name)
	if carefinderURL == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	rpi := c.reverseProxy.Load()
	if rpi == nil || rpi.(*reverseProxyInfo).currentURL != carefinderURL {
		p, err := url.Parse(carefinderURL)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		rpi = &reverseProxyInfo{
			reverseProxy: nethttputil.NewSingleHostReverseProxy(p),
			currentURL:   carefinderURL,
		}
		c.reverseProxy.Store(rpi)
	}

	rpi.(*reverseProxyInfo).reverseProxy.ServeHTTP(w, r)
}
