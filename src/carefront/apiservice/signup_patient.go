package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/golog"
	"carefront/libs/maps"
	thriftapi "carefront/thrift/api"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/schema"
)

type SignupPatientHandler struct {
	DataApi api.DataAPI
	AuthApi thriftapi.Auth
	MapsApi maps.MapsService
}

type PatientSignedupResponse struct {
	Token   string          `json:"token"`
	Patient *common.Patient `json:"patient,omitempty"`
}

func (s *SignupPatientHandler) NonAuthenticated() bool {
	return true
}

type SignupPatientRequestData struct {
	Email      string `schema:"email,required"`
	Password   string `schema:"password,required"`
	FirstName  string `schema:"first_name,required"`
	LastName   string `schema:"last_name,required"`
	Dob        string `schema:"dob,required"`
	Gender     string `schema:"gender,required"`
	Zipcode    string `schema:"zip_code,required"`
	Phone      string `schema:"phone,required"`
	Agreements string `schema:"agreements"`
	DoctorId   int64  `schema:"doctor_id"`
}

func (s *SignupPatientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData SignupPatientRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
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
	res, err := s.AuthApi.SignUp(requestData.Email, requestData.Password)
	if _, ok := err.(*thriftapi.LoginAlreadyExists); ok {
		WriteUserError(w, http.StatusBadRequest, "An account with the specified email address already exists.")
		return
	}

	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Internal Servier Error. Unable to register patient")
		return
	}

	// ignore the error case of the reverse geocoding failing because it is not detrimental to
	// serving the patient, especially after the client has already checked to ensure that we can actually
	// serve the patient.
	cityStateInfo, _ := s.MapsApi.ConvertZipcodeToCityState(requestData.Zipcode)

	// then, register the signed up user as a patient
	patient, err := s.DataApi.RegisterPatient(res.AccountId, requestData.FirstName, requestData.LastName, requestData.Gender, requestData.Zipcode, cityStateInfo.LongCityName, cityStateInfo.ShortStateName, requestData.Phone, api.PATIENT_PHONE_CELL, time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC))
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to register patient: "+err.Error())
		return
	}

	// track patient agreements
	if requestData.Agreements != "" {
		patientAgreements := make(map[string]bool)
		for _, agreement := range strings.Split(requestData.Agreements, ",") {
			patientAgreements[strings.TrimSpace(agreement)] = true
		}

		err = s.DataApi.TrackPatientAgreements(patient.PatientId.Int64(), patientAgreements)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to track patient agreements: "+err.Error())
			return
		}
	}

	// create care team for patient
	if requestData.DoctorId != 0 {
		_, err = s.DataApi.CreateCareTeamForPatientWithPrimaryDoctor(patient.PatientId.Int64(), requestData.DoctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create care team with specified doctor for patient: "+err.Error())
			return
		}
	} else {
		_, err = s.DataApi.CreateCareTeamForPatient(patient.PatientId.Int64())
		if err != nil {
			golog.Errorf(err.Error())
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create care team for patient :"+err.Error())
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientSignedupResponse{Token: res.Token, Patient: patient})
}
