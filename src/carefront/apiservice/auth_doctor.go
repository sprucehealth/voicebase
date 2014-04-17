package apiservice

import (
	"carefront/api"
	"carefront/common"
	thriftapi "carefront/thrift/api"
	"net/http"

	"github.com/gorilla/schema"
)

type DoctorAuthenticationHandler struct {
	AuthApi thriftapi.Auth
	DataApi api.DataAPI
}

type DoctorAuthenticationResponse struct {
	Token  string         `json:"token,omitempty"`
	Doctor *common.Doctor `json:"doctor,omitempty"`
}

func (d *DoctorAuthenticationHandler) NonAuthenticated() bool {
	return true
}

type DoctorAuthenticationRequestData struct {
	Email    string `schema:"email,required"`
	Password string `schema:"password,required"`
}

func (d *DoctorAuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData DoctorAuthenticationRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
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
		doctor, err := d.DataApi.GetDoctorFromAccountId(res.AccountId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor id from account id "+err.Error())
			return
		}

		WriteJSONToHTTPResponseWriter(w, http.StatusOK, DoctorAuthenticationResponse{Token: res.Token, Doctor: doctor})
	}

}
