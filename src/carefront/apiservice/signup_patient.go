package apiservice

import (
	"carefront/api"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SignupPatientHandler struct {
	DataApi api.DataAPI
	AuthApi api.Auth
}

type PatientSignedupResponse struct {
	Token     string `json:"token"`
	PatientId int64  `json:"patientId,string"`
}

type PatientSignupErrorResponse struct {
	ErrorString string `json:"error"`
}

func (s *SignupPatientHandler) NonAuthenticated() bool {
	return true
}

func (s *SignupPatientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	dob := r.FormValue("dob")
	gender := r.FormValue("gender")
	zipCode := r.FormValue("zip_code")

	if email == "" || password == "" || firstName == "" || lastName == "" || dob == "" || gender == "" || zipCode == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// ensure that the date of birth can be correctly parsed
	// Note that the date will be returned as MM/DD/YYYY
	dobParts := strings.Split(dob, "/")

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
	token, accountId, err := s.AuthApi.Signup(email, password)
	if err == api.ErrSignupFailedUserExists {
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(w, PatientSignupErrorResponse{err.Error()})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		WriteJSONToHTTPResponseWriter(w, PatientSignupErrorResponse{err.Error()})
		return
	}

	// then, register the signed up user as a patient
	patientId, err := s.DataApi.RegisterPatient(accountId, firstName, lastName, gender, zipCode, time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC))
	err = WriteJSONToHTTPResponseWriter(w, PatientSignedupResponse{token, patientId})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
