package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
)

const (
	VerifyAuthCode = 3198456
	authorization  = "authorization"
	authentication = "authentication"
)

func verifyAuthSetupInTest(ctx context.Context, w http.ResponseWriter, r *http.Request, h httputil.ContextHandler, action string, response int) bool {
	if environment.IsTest() {
		test := r.FormValue("test")
		if test != "" && test == action {
			httputil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"result": response,
			})
			return true
		} else if test != "" {
			// bypass the check in the handler if the test parameter
			// value does not match the intended action. This is so that
			// any request handlers deeper in the chain can handle the test
			// probe appropriately
			h.ServeHTTP(ctx, w, r)
			return true
		}
	}
	return false
}
