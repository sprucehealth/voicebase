package apiservice

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/third_party/github.com/dchest/validator"
)

type signupDoctorHandler struct {
	dataAPI     api.DataAPI
	authAPI     api.AuthAPI
	environment string
}

func NewSignupDoctorHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, environment string) *signupDoctorHandler {
	return &signupDoctorHandler{
		dataAPI:     dataAPI,
		authAPI:     authAPI,
		environment: environment,
	}
}

type DoctorSignedupResponse struct {
	Token    string `json:"token"`
	DoctorId int64  `json:"doctorId,string"`
	PersonId int64  `json:"person_id,string"`
}

func (d *signupDoctorHandler) NonAuthenticated() bool {
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

func (d *signupDoctorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var requestData SignupDoctorRequestData
	if err := DecodeRequestData(&requestData, r); err != nil {
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
	accountID, token, err := d.authAPI.SignUp(requestData.Email, requestData.Password, api.DOCTOR_ROLE)
	if err == api.LoginAlreadyExists {
		WriteUserError(w, http.StatusBadRequest, "An account with the specified email address already exists.")
		return
	} else if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Internal Servier Error. Unable to register doctor: "+err.Error())
		return
	}

	doctorToRegister := &common.Doctor{
		AccountId:           encoding.NewObjectId(accountID),
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

	// then, register the signed up user as a doctor
	doctorId, err := d.dataAPI.RegisterDoctor(doctorToRegister)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Something went wrong when trying to sign up doctor: "+err.Error())
		return
	}

	// only add the doctor as being eligible in CA for non-prod environments
	if d.environment != "prod" {

		careProvidingStateId, err := d.dataAPI.GetCareProvidingStateId("CA", HEALTH_CONDITION_ACNE_ID)
		if err != nil {
			WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateId, doctorId); err != nil {
			WriteError(err, w, r)
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorSignedupResponse{
		Token:    token,
		DoctorId: doctorId,
		PersonId: doctorToRegister.PersonId,
	})
}
