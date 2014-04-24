package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/golog"
	thriftapi "carefront/thrift/api"
	"net/http"
	"strings"

	"github.com/gorilla/schema"
)

type SignupPatientHandler struct {
	DataApi api.DataAPI
	AuthApi thriftapi.Auth
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
	dobParts := strings.Split(requestData.Dob, encoding.DOB_SEPARATOR)

	if len(dobParts) < 3 {
		WriteUserError(w, http.StatusBadRequest, "Unable to parse dob. Format should be "+encoding.DOB_FORMAT)
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

	newPatient := &common.Patient{
		AccountId: encoding.NewObjectId(res.AccountId),
		FirstName: requestData.FirstName,
		LastName:  requestData.LastName,
		Gender:    requestData.Gender,
		ZipCode:   requestData.Zipcode,
		PhoneNumbers: []*common.PhoneInformation{&common.PhoneInformation{
			Phone:     requestData.Phone,
			PhoneType: api.PHONE_CELL,
		},
		},
	}

	newPatient.Dob, err = encoding.NewDobFromComponents(dobParts[0], dobParts[1], dobParts[2])
	if err != nil {
		WriteUserError(w, http.StatusBadRequest, "Unable to parse date of birth. Required format + "+encoding.DOB_FORMAT)
		return
	}

	// then, register the signed up user as a patient
	err = s.DataApi.RegisterPatient(newPatient)
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

		err = s.DataApi.TrackPatientAgreements(newPatient.PatientId.Int64(), patientAgreements)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to track patient agreements: "+err.Error())
			return
		}
	}

	// create care team for patient
	if requestData.DoctorId != 0 {
		_, err = s.DataApi.CreateCareTeamForPatientWithPrimaryDoctor(newPatient.PatientId.Int64(), requestData.DoctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create care team with specified doctor for patient: "+err.Error())
			return
		}
	} else {
		_, err = s.DataApi.CreateCareTeamForPatient(newPatient.PatientId.Int64())
		if err != nil {
			golog.Errorf(err.Error())
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create care team for patient :"+err.Error())
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientSignedupResponse{Token: res.Token, Patient: newPatient})
}
