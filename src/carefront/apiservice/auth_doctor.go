package apiservice

import (
	"carefront/api"
	thriftapi "carefront/thrift/api"
	"net/http"

	"github.com/gorilla/schema"
)

type DoctorAuthenticationHandler struct {
	AuthApi thriftapi.Auth
	DataApi api.DataAPI
}

type DoctorAuthenticationResponse struct {
	Token    string `json:"token"`
	DoctorId int64  `json:"doctorId,string"`
}

func (d *DoctorAuthenticationHandler) NonAuthenticated() bool {
	return true
}

type DoctorAuthenticationRequestData struct {
	Email    string `schema:"email,required"`
	Password string `schema:"password,required"`
}

func (d *DoctorAuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(DoctorAuthenticationRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	if res, err := d.AuthApi.LogIn(requestData.Email, requestData.Password); err != nil {
		switch err.(type) {
		case *thriftapi.NoSuchLogin, *thriftapi.InvalidPassword:
			WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
			return
		default:
			WriteDeveloperError(w, http.StatusInternalServerError, "Internal server error: "+err.Error())
			return
		}
	} else {
		doctorId, err := d.DataApi.GetDoctorIdFromAccountId(res.AccountId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor id from account id "+err.Error())
			return
		}
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, DoctorAuthenticationResponse{Token: res.Token, DoctorId: doctorId})
	}

}
