package doctor

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/third_party/github.com/dchest/validator"
)

type signupDoctorHandler struct {
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

func NewSignupDoctorHandler(dataAPI api.DataAPI, authAPI api.AuthAPI) http.Handler {
	return apiservice.SupportedMethods(&signupDoctorHandler{
		dataAPI: dataAPI,
		authAPI: authAPI,
	}, []string{apiservice.HTTP_POST})
}

type DoctorSignedupResponse struct {
	Token    string `json:"token"`
	DoctorId int64  `json:"doctorId,string"`
	PersonId int64  `json:"person_id,string"`
}

func (d *signupDoctorHandler) NonAuthenticated() bool {
	return true
}

func (d *signupDoctorHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

type SignupDoctorRequestData struct {
	Email            string `schema:"email,required"`
	Password         string `schema:"password,required"`
	FirstName        string `schema:"first_name,required"`
	LastName         string `schema:"last_name,required"`
	MiddleName       string `schema:"middle_name"`
	ShortTitle       string `schema:"short_title"`
	LongTitle        string `schema:"long_title"`
	ShortDisplayName string `schema:"short_display_name"`
	LongDisplayName  string `schema:"long_display_name"`
	Suffix           string `schema:"suffix"`
	Prefix           string `schema:"prefix"`
	DOB              string `schema:"dob,required"`
	Gender           string `schema:"gender,required"`
	ClinicianId      int64  `schema:"clinician_id,required"`
	AddressLine1     string `schema:"address_line_1,required"`
	AddressLine2     string `schema:"address_line_2"`
	City             string `schema:"city"`
	State            string `schema:"state"`
	ZipCode          string `schema:"zip_code"`
	Phone            string `schema:"phone,required"`
}

func (d *signupDoctorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var requestData SignupDoctorRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return

	}
	if !validator.IsValidEmail(requestData.Email) {
		apiservice.WriteValidationError("Please enter a valid email address", w, r)
		return
	}

	// ensure that the date of birth can be correctly parsed
	// Note that the date will be returned as MM/DD/YYYY
	dobParts := strings.Split(requestData.DOB, encoding.DOBSeparator)
	if len(dobParts) != 3 {
		apiservice.WriteValidationError("DOB not valid. Required format "+encoding.DOBFormat, w, r)
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
		apiservice.WriteValidationError("An account with the specified email address already exists.", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctorToRegister := &common.Doctor{
		AccountId:           encoding.NewObjectId(accountID),
		FirstName:           requestData.FirstName,
		LastName:            requestData.LastName,
		Gender:              requestData.Gender,
		ShortTitle:          requestData.ShortTitle,
		LongTitle:           requestData.LongTitle,
		ShortDisplayName:    requestData.ShortDisplayName,
		LongDisplayName:     requestData.LongDisplayName,
		Suffix:              requestData.Suffix,
		Prefix:              requestData.Prefix,
		MiddleName:          requestData.MiddleName,
		DOB:                 encoding.DOB{Year: year, Month: month, Day: day},
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
		apiservice.WriteError(err, w, r)
		return
	}

	// only add the doctor as being eligible in CA for non-prod environments
	if !environment.IsProd() {

		careProvidingStateId, err := d.dataAPI.GetCareProvidingStateId("CA", apiservice.HEALTH_CONDITION_ACNE_ID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateId, doctorId); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSON(w, &DoctorSignedupResponse{
		Token:    token,
		DoctorId: doctorId,
		PersonId: doctorToRegister.PersonId,
	})
}
