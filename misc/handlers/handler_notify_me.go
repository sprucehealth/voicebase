package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type notifyMeHandler struct {
	dataAPI api.DataAPI
}

type notifyMeRequest struct {
	Email string `json:"email"`
	State string `json:"state_code"`
}

func NewNotifyMeHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&notifyMeHandler{
			dataAPI: dataAPI,
		}), httputil.Post, httputil.Put)
}

func (n *notifyMeHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var rd notifyMeRequest
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteBadRequestError(ctx, err, w, r)
		return
	}

	_, stateCode, err := n.dataAPI.State(rd.State)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	spruceHeaders := apiservice.ExtractSpruceHeaders(r)
	if err := n.dataAPI.RecordForm(&common.NotifyMeForm{
		Email:     rd.Email,
		State:     stateCode,
		Platform:  spruceHeaders.Platform.String(),
		UniqueKey: spruceHeaders.DeviceID,
	}, "mobile", httputil.RequestID(ctx)); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
