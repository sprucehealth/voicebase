package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/golog"
	"net/http"
	"strconv"
	"strings"

	"github.com/dchest/validator"
	"github.com/gorilla/schema"
)

type SignupDoctorHandler struct {
	DataApi api.DataAPI
	AuthApi api.AuthAPI
}

type DoctorSignedupResponse struct {
	Token    string `json:"token"`
	DoctorId int64  `json:"doctorId,string"`
	PersonId int64  `json:"person_id,string"`
}

func (d *SignupDoctorHandler) NonAuthenticated() bool {
	return true
}

type SignupDoctorRequestData struct {
	Email        string `schema:"email,required"`
	Password     string `schema:"password,required"`
	FirstName    string `schema:"first_name,required"`
	LastName     string `schema:"last_name,required"`
	Dob          string `schema:"dob,required"`
	Gender       string `schema:"gender,required"`
	ClinicianId  int64  `schema:"clinician_id,required"`
	AddressLine1 string `schema:"address_line_1,required"`
	AddressLine2 string `schema:"address_line_2"`
	City         string `schema:"city"`
	State        string `schema:"state"`
	ZipCode      string `schema:"zip_code"`
	Phone        string `schema:"phone,required"`
}

func (d *SignupDoctorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData SignupDoctorRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input to signup doctor: "+err.Error())
		return
	}

	if !validator.IsValidEmail(requestData.Email) {
		WriteUserError(w, http.StatusBadRequest, "Please enter a valid email address")
		golog.Infof("Invalid email during doctor signup: %s", requestData.Email)
		return
	}

	// ensure that the date of birth can be correctly parsed
	// Note that the date will be returned as MM/DD/YYYY
	dobParts := strings.Split(requestData.Dob, encoding.DOB_SEPARATOR)
	if len(dobParts) != 3 {
		WriteUserError(w, http.StatusBadRequest, "Dob not valid. Required format "+encoding.DOB_FORMAT)
		return
	}

	year, err := strconv.Atoi(dobParts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	month, err := strconv.Atoi(dobParts[1])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	day, err := strconv.Atoi(dobParts[2])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// first, create an account for the user
	res, err := d.AuthApi.SignUp(requestData.Email, requestData.Password, api.DOCTOR_ROLE)
	if err == api.LoginAlreadyExists {
		WriteUserError(w, http.StatusBadRequest, "An account with the specified email address already exists.")
		return
	}

	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Internal Servier Error. Unable to register doctor: "+err.Error())
		return
	}

	doctorToRegister := &common.Doctor{
		AccountId:           encoding.NewObjectId(res.AccountId),
		FirstName:           requestData.FirstName,
		LastName:            requestData.LastName,
		Gender:              requestData.Gender,
		Dob:                 encoding.Dob{Year: year, Month: month, Day: day},
		CellPhone:           requestData.Phone,
		DoseSpotClinicianId: requestData.ClinicianId,
		DoctorAddress: &common.Address{
			AddressLine1: requestData.AddressLine1,
			AddressLine2: requestData.AddressLine2,
			City:         requestData.City,
			State:        requestData.State,
			ZipCode:      requestData.ZipCode,
		},
		PromptStatus: common.Unprompted,
	}

	// then, register the signed up user as a patient
	doctorId, err := d.DataApi.RegisterDoctor(doctorToRegister)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Something went wrong when trying to sign up doctor: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorSignedupResponse{
		Token:    res.Token,
		DoctorId: doctorId,
		PersonId: doctorToRegister.PersonId,
	})
}
