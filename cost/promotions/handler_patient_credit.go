package promotions

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

type creditsHandler struct {
	dataAPI api.DataAPI
}

type creditsRequestData struct {
	PatientID int64 `json:"patient_id,string"`
	Credit    int   `json:"credit"`
}

func NewPatientCreditsHandler(dataAPI api.DataAPI) http.Handler {
	return &creditsHandler{
		dataAPI: dataAPI,
	}
}

func (c *creditsHandler) IsAuthorized(r *http.Request) (bool, error) {

	if r.Method != apiservice.HTTP_PUT {
		return false, apiservice.NewAccessForbiddenError()
	}

	if apiservice.GetContext(r).Role != api.ADMIN_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (c *creditsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rd creditsRequestData

	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	if err := c.dataAPI.UpdateCredit(rd.PatientID, rd.Credit, USDUnit.String()); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)

}
