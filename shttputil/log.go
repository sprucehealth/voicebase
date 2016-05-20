package shttputil

import (
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/trace/tracectx"
	"golang.org/x/net/context"
)

// WebRequestLogger returns a httputil.LoggingHandler compatible function that performs common Spruce web request logging
func WebRequestLogger(behindProxy bool) func(ctx context.Context, ev *httputil.RequestEvent) {
	return func(ctx context.Context, ev *httputil.RequestEvent) {
		contextVals := []interface{}{
			"Method", ev.Request.Method,
			"URL", ev.URL.String(),
			"UserAgent", ev.Request.UserAgent(),
			"RequestID", tracectx.RequestID(ctx),
			"RemoteAddr", httputil.RemoteAddrFromRequest(ev.Request, behindProxy),
			"StatusCode", ev.StatusCode,
		}

		log := golog.Context(contextVals...)

		if ev.Panic != nil {
			log.Criticalf("http: panic: %v\n%s", ev.Panic, ev.StackTrace)
		}
	}
}
