package notify

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type promptStatusHandler struct {
	dataAPI api.DataAPI
}

func NewPromptStatusHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&promptStatusHandler{
				dataAPI: dataAPI,
			}), httputil.Put)
}

type promptStatusRequestData struct {
	PromptStatus string `schema:"prompt_status" json:"prompt_status"`
}

func (p *promptStatusHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	rData := &promptStatusRequestData{}
	if err := apiservice.DecodeRequestData(rData, r); err != nil {
		apiservice.WriteBadRequestError(ctx, err, w, r)
		return
	}

	pStatus, err := common.ParsePushPromptStatus(rData.PromptStatus)
	if err != nil {
		apiservice.WriteValidationError(ctx, "Invalid prompt_status", w, r)
		return
	}

	if err := p.dataAPI.SetPushPromptStatus(apiservice.MustCtxAccount(ctx).ID, pStatus); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
