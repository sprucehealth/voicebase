package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

const (
	BILLING_ADDRESS_TYPE = "BILLING"
)

type addressHandler struct {
	dataAPI     api.DataAPI
	addressType string
}

func NewAddressHandler(dataAPI api.DataAPI, addressType string) http.Handler {
	return &addressHandler{
		dataAPI:     dataAPI,
		addressType: addressType,
	}
}

type UpdatePatientAddressRequestData struct {
	AddressLine1 string `schema:"address_line_1,required"`
	AddressLine2 string `schema:"address_line_2"`
	City         string `schema:"city,required"`
	State        string `schema:"state,required"`
	Zipcode      string `schema:"zip_code,required"`
}

func (u *addressHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

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

	patientId, err := u.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	err = u.dataAPI.UpdatePatientAddress(patientId, requestData.AddressLine1, requestData.AddressLine2, requestData.City, requestData.State, requestData.Zipcode, u.addressType)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
