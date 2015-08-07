package notify

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type promptStatusHandler struct {
	dataAPI api.DataAPI
}

func NewPromptStatusHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&promptStatusHandler{
				dataAPI: dataAPI,
			}), httputil.Put)
}

type promptStatusRequestData struct {
	PromptStatus string `schema:"prompt_status" json:"prompt_status"`
}

func (p *promptStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rData := &promptStatusRequestData{}
	if err := apiservice.DecodeRequestData(rData, r); err != nil {
		apiservice.WriteBadRequestError(err, w, r)
		return
	}

	pStatus, err := common.ParsePushPromptStatus(rData.PromptStatus)
	if err != nil {
		apiservice.WriteValidationError("Invalid prompt_status", w, r)
		return
	}

	if err := p.dataAPI.SetPushPromptStatus(apiservice.GetContext(r).AccountID, pStatus); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
