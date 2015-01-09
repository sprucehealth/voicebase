package ratelimit

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/golog"
)

func Handler(h http.Handler, rl RateLimiter, statsRegistry metrics.Registry) http.Handler {
	statRateLimited := metrics.NewCounter()
	statNotRateLimited := metrics.NewCounter()
	statsRegistry.Add("ratelimited", statRateLimited)
	statsRegistry.Add("successful", statNotRateLimited)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, err := rl.Check(1); err != nil {
			golog.Errorf("Rate limit check failed: %s", err.Error())
		} else if !ok {
			statRateLimited.Inc(1)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func RemoteAddrHandler(h http.Handler, rl KeyedRateLimiter, prefix string, statsRegistry metrics.Registry) http.Handler {
	statRateLimited := metrics.NewCounter()
	statsRegistry.Add("ratelimited", statRateLimited)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, err := rl.Check(prefix+r.RemoteAddr, 1); err != nil {
			golog.Errorf("Rate limit check failed: %s", err.Error())
		} else if !ok {
			statRateLimited.Inc(1)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}
