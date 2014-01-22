package apiservice

import (
	"carefront/api"
	thriftapi "carefront/thrift/api"
	"github.com/gorilla/schema"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SignupDoctorHandler struct {
	DataApi api.DataAPI
	AuthApi thriftapi.Auth
}

type DoctorSignedupResponse struct {
	Token    string `json:"token"`
	DoctorId int64  `json:"doctorId, string"`
}

func (d *SignupDoctorHandler) NonAuthenticated() bool {
	return true
}

type SignupDoctorRequestData struct {
	Email     string `schema:"email,required"`
	Password  string `schema:"password,required"`
	FirstName string `schema:"first_name,required"`
	LastName  string `schema:"last_name,required"`
	Dob       string `schema:"dob,required"`
	Gender    string `schema:"gender,required"`
}

func (d *SignupDoctorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(SignupDoctorRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input to signup doctor: "+err.Error())
		return
	}
	// ensure that the date of birth can be correctly parsed
	// Note that the date will be returned as MM/DD/YYYY
	dobParts := strings.Split(requestData.Dob, "/")

	month, err := strconv.Atoi(dobParts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	day, err := strconv.Atoi(dobParts[1])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	year, err := strconv.Atoi(dobParts[2])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// first, create an account for the user
	res, err := d.AuthApi.SignUp(requestData.Email, requestData.Password)
	if _, ok := err.(*thriftapi.LoginAlreadyExists); ok {
		WriteUserError(w, http.StatusBadRequest, "An account with the specified email address already exists.")
		return
	}

	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Internal Servier Error. Unable to register doctor: "+err.Error())
		return
	}

	// then, register the signed up user as a patient
	doctorId, err := d.DataApi.RegisterDoctor(res.AccountId, requestData.FirstName, requestData.LastName, requestData.Gender, time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC))
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Something went wrong when trying to sign up doctor: "+err.Error())
		return
	}
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, DoctorSignedupResponse{res.Token, doctorId})

}
