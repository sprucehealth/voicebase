package doctor

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
)

type signupDoctorHandler struct {
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

func NewSignupDoctorHandler(dataAPI api.DataAPI, authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&signupDoctorHandler{
			dataAPI: dataAPI,
			authAPI: authAPI,
		}), []string{"POST"})
}

type DoctorSignedupResponse struct {
	Token    string `json:"token"`
	DoctorID int64  `json:"doctorId,string"`
	PersonID int64  `json:"person_id,string"`
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
	ClinicianID      int64  `schema:"clinician_id,required"`
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
	if !email.IsValidEmail(requestData.Email) {
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
	accountID, err := d.authAPI.CreateAccount(requestData.Email, requestData.Password, api.DOCTOR_ROLE)
	if err == api.LoginAlreadyExists {
		apiservice.WriteValidationError("An account with the specified email address already exists.", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctorToRegister := &common.Doctor{
		AccountID:           encoding.NewObjectID(accountID),
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
		DoseSpotClinicianID: requestData.ClinicianID,
		DoctorAddress: &common.Address{
			AddressLine1: requestData.AddressLine1,
			AddressLine2: requestData.AddressLine2,
			City:         requestData.City,
			State:        requestData.State,
			ZipCode:      requestData.ZipCode,
		},
		PromptStatus: common.Unprompted,
	}

	doctorToRegister.CellPhone, err = common.ParsePhone(requestData.Phone)
	if err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// then, register the signed up user as a doctor
	doctorID, err := d.dataAPI.RegisterDoctor(doctorToRegister)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// only add the doctor as being eligible in CA for non-prod environments
	if !environment.IsProd() {
		// TODO: don't assume acne
		careProvidingStateID, err := d.dataAPI.GetCareProvidingStateID("CA", api.AcnePathwayTag)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, doctorID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	token, err := d.authAPI.CreateToken(accountID, api.Mobile, api.RegularAuth)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &DoctorSignedupResponse{
		Token:    token,
		DoctorID: doctorID,
		PersonID: doctorToRegister.PersonID,
	})
}
