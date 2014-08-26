package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
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

func (d *doctorAuthenticationHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}
	return true, nil
}

func (d *doctorAuthenticationHandler) NonAuthenticated() bool {
	return true
}

type DoctorAuthenticationRequestData struct {
	Email    string `schema:"email,required"`
	Password string `schema:"password,required"`
}

func (d *doctorAuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var requestData DoctorAuthenticationRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	account, err := d.authAPI.Authenticate(requestData.Email, requestData.Password, api.Mobile, api.RegularAuth)
	if err != nil {
		switch err {
		case api.LoginDoesNotExist, api.InvalidPassword:
			apiservice.WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
			return
		}
		apiservice.WriteError(err, w, r)
		return
	}
	token, err := d.authAPI.CreateToken(account.ID, api.Mobile)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctor, err := d.dataAPI.GetDoctorFromAccountId(account.ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, DoctorAuthenticationResponse{Token: token, Doctor: doctor})
}
