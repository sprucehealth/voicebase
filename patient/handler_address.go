package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

const (
	BILLING_ADDRESS_TYPE = "BILLING"
)

type addressHandler struct {
	dataAPI     api.DataAPI
	addressType string
}

func NewAddressHandler(dataAPI api.DataAPI, addressType string) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&addressHandler{
				dataAPI:     dataAPI,
				addressType: addressType,
			}), []string{"POST"})
}

type UpdatePatientAddressRequestData struct {
	AddressLine1 string `schema:"address_line_1,required"`
	AddressLine2 string `schema:"address_line_2"`
	City         string `schema:"city,required"`
	State        string `schema:"state,required"`
	ZipCode      string `schema:"zip_code,required"`
}

func (u *addressHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

func (u *addressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var requestData UpdatePatientAddressRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	patientID, err := u.dataAPI.GetPatientIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	err = u.dataAPI.UpdatePatientAddress(patientID, requestData.AddressLine1, requestData.AddressLine2,
		requestData.City, requestData.State, requestData.ZipCode, u.addressType)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
