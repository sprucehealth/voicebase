package promotions

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type creditsHandler struct {
	dataAPI api.DataAPI
}

type creditsRequestData struct {
	AccountID int64 `json:"account_id,string"`
	Credit    int   `json:"credit"`
}

// NewPatientCreditsHandler returns a new initialzed instance of the creditsHandler
func NewPatientCreditsHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&creditsHandler{dataAPI: dataAPI}),
			api.RoleAdmin),
		httputil.Put)
}

func (c *creditsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var rd creditsRequestData

	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	if err := c.dataAPI.UpdateCredit(rd.AccountID, rd.Credit, USDUnit.String()); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
