package apiservice

import (
	"carefront/api"
	"github.com/gorilla/schema"
	"net/http"
)

const (
	BILLING_ADDRESS_TYPE = "BILLING"
)

type UpdatePatientAddressHandler struct {
	DataApi     api.DataAPI
	AddressType string
}

type UpdatePatientAddressRequestData struct {
	AddressLine1 string `schema:"address_line_1,required"`
	AddressLine2 string `schema:"address_line_2"`
	City         string `schema:"city,required"`
	State        string `schema:"state,required"`
	Zipcode      string `schema:"zip_code,required"`
}

func (u *UpdatePatientAddressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		u.updatePatientAddress(w, r)
	default:
		WriteJSONToHTTPResponseWriter(w, http.StatusNotFound, nil)
	}
}

func (u *UpdatePatientAddressHandler) updatePatientAddress(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(UpdatePatientAddressRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientId, err := u.DataApi.GetPatientIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusOK, "Unable to get patient id from account id: "+err.Error())
		return
	}

	err = u.DataApi.UpdatePatientAddress(patientId, requestData.AddressLine1, requestData.AddressLine2, requestData.City, requestData.State, requestData.Zipcode, u.AddressType)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update patient address: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}
