package ratelimit

import (
	"net/http"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

func Handler(h httputil.ContextHandler, rl RateLimiter, statsRegistry metrics.Registry) httputil.ContextHandler {
	statRateLimited := metrics.NewCounter()
	statNotRateLimited := metrics.NewCounter()
	statsRegistry.Add("ratelimited", statRateLimited)
	statsRegistry.Add("successful", statNotRateLimited)
	return httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if ok, err := rl.Check(1); err != nil {
			golog.Errorf("Rate limit check failed: %s", err.Error())
		} else if !ok {
			statRateLimited.Inc(1)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h.ServeHTTP(ctx, w, r)
	})
}

func RemoteAddrHandler(h httputil.ContextHandler, rl KeyedRateLimiter, prefix string, statsRegistry metrics.Registry) httputil.ContextHandler {
	statRateLimited := metrics.NewCounter()
	statsRegistry.Add("ratelimited", statRateLimited)
	return httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if ok, err := rl.Check(prefix+r.RemoteAddr, 1); err != nil {
			golog.Errorf("Rate limit check failed: %s", err.Error())
		} else if !ok {
			statRateLimited.Inc(1)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h.ServeHTTP(ctx, w, r)
	})
}
