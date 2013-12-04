package apiservice

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"carefront/api"
	"carefront/thriftapi"
	"github.com/gorilla/schema"
)

type SignupPatientHandler struct {
	DataApi api.DataAPI
	AuthApi thriftapi.Auth
}

type PatientSignedupResponse struct {
	Token     string `json:"token"`
	PatientId int64  `json:"patientId,string"`
}

func (s *SignupPatientHandler) NonAuthenticated() bool {
	return true
}

type SignupPatientRequestData struct {
	Email     string `schema:"email,required"`
	Password  string `schema:"password,required"`
	FirstName string `schema:"first_name,required"`
	LastName  string `schema:"last_name,required"`
	Dob       string `schema:"dob,required"`
	Gender    string `schema:"gender,required"`
	Zipcode   string `schema:"zip_code,required"`
}

func (s *SignupPatientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(SignupPatientRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
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
	res, err := s.AuthApi.Signup(requestData.Email, requestData.Password)
	if _, ok := err.(*thriftapi.LoginAlreadyExists); ok {
		WriteUserError(w, http.StatusBadRequest, "An account with the specified email address already exists.")
		return
	}

	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Internal Servier Error. Unable to register patient")
		return
	}

	// then, register the signed up user as a patient
	patientId, err := s.DataApi.RegisterPatient(res.AccountId, requestData.FirstName, requestData.LastName, requestData.Gender, requestData.Zipcode, time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC))
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientSignedupResponse{res.Token, patientId})
}
