package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

type meHandler struct {
	dataAPI api.DataAPI
}

func NewMeHandler(dataAPI api.DataAPI) http.Handler {
	return &meHandler{
		dataAPI: dataAPI,
	}
}

func (m *meHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.PATIENT_ROLE {
		return false, nil
	}

	return true, nil
}

func (m *meHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

	patient, err := m.dataAPI.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{"patient": patient})
}
