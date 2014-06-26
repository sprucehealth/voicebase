package notify

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"net/http"
)

type promptStatusHandler struct {
	dataApi api.DataAPI
}

func NewPromptStatusHandler(dataApi api.DataAPI) *promptStatusHandler {
	return &promptStatusHandler{
		dataApi: dataApi,
	}
}

type promptStatusRequestData struct {
	PromptStatus string `schema:"prompt_status"`
}

func (p *promptStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_PUT {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	rData := &promptStatusRequestData{}
	if err := apiservice.DecodeRequestData(rData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	pStatus, err := common.GetPushPromptStatus(rData.PromptStatus)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := p.dataApi.SetPushPromptStatus(apiservice.GetContext(r).AccountId, pStatus); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}
