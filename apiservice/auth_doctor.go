package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
)

type doctorAuthenticationHandler struct {
	authAPI api.AuthAPI
	dataAPI api.DataAPI
}

type DoctorAuthenticationResponse struct {
	Token  string         `json:"token,omitempty"`
	Doctor *common.Doctor `json:"doctor,omitempty"`
}

func NewDoctorAuthenticationHandler(dataAPI api.DataAPI, authAPI api.AuthAPI) http.Handler {
	return &doctorAuthenticationHandler{
		dataAPI: dataAPI,
		authAPI: authAPI,
	}
}

func (d *doctorAuthenticationHandler) NonAuthenticated() bool {
	return true
}

type DoctorAuthenticationRequestData struct {
	Email    string `schema:"email,required"`
	Password string `schema:"password,required"`
}

func (d *doctorAuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	if account, token, err := d.authAPI.LogIn(requestData.Email, requestData.Password); err != nil {
		switch err {
		case api.LoginDoesNotExist, api.InvalidPassword:
			WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
			return
		default:
			WriteDeveloperError(w, http.StatusInternalServerError, "Internal server error: "+err.Error())
			return
		}
	} else {
		doctor, err := d.dataAPI.GetDoctorFromAccountId(account.ID)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor id from account id "+err.Error())
			return
		}

		WriteJSONToHTTPResponseWriter(w, http.StatusOK, DoctorAuthenticationResponse{Token: token, Doctor: doctor})
	}
}
