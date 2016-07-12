package handlers

import (
	"net/http"

	"context"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
)

type appleDeeplinkHandler struct {
}

type appDetails struct {
	AppID string   `json:"appID"`
	Paths []string `json:"paths"`
}
type appLinks struct {
	Apps    []interface{} `json:"apps"`
	Details []appDetails  `json:"details"`
}

type appLinksContainer struct {
	AppLinks appLinks `json:"appLinks"`
}

func (*appleDeeplinkHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	switch environment.GetCurrent() {
	case environment.Prod:
		httputil.JSONResponse(w, http.StatusOK, appLinksContainer{
			AppLinks: appLinks{
				Apps: []interface{}{},
				Details: []appDetails{
					{
						AppID: "VASUED9B9G.com.sprucehealth.messenger-live",
						Paths: []string{"*"},
					},
				},
			},
		})

		return
	case environment.Staging:
		httputil.JSONResponse(w, http.StatusOK, appLinksContainer{
			AppLinks: appLinks{
				Apps: []interface{}{},
				Details: []appDetails{
					{
						AppID: "PRKG6MA37R.com.sprucehealth.messenger-staging",
						Paths: []string{"*"},
					},
				},
			},
		})
		return
	}
}
