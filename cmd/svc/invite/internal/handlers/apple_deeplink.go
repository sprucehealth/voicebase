package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type appleDeeplinkHandler struct {
}

func (*appleDeeplinkHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	switch environment.GetCurrent() {
	case environment.Prod:
		httputil.JSONResponse(w, http.StatusOK, `{
      "applinks": {
        "apps": [],
        "details": [
          {
            "appID": "VASUED9B9G.com.sprucehealth.messenger-live",
            "paths": [
              "*"
            ]
          }
        ]
      }
    }`)
		return
	case environment.Staging:
		httputil.JSONResponse(w, http.StatusOK, `{
      "applinks": {
        "apps": [],
        "details": [
          {
            "appID": "PRKG6MA37R.com.sprucehealth.messenger-staging",
            "paths": [
              "*"
            ]
          }
        ]
      }
    }`)
		return
	}
}
