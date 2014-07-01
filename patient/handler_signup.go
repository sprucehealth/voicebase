package patient

import (
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/third_party/github.com/dchest/validator"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/schema"
)

type SignupHandler struct {
	dataApi    api.DataAPI
	authApi    api.AuthAPI
	addressAPI address.AddressValidationAPI
}

type PatientSignedupResponse struct {
	Token   string          `json:"token"`
	Patient *common.Patient `json:"patient,omitempty"`
}

func (s *SignupHandler) NonAuthenticated() bool {
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

func NewSignupHandler(dataApi api.DataAPI, authApi api.AuthAPI, addressAPI address.AddressValidationAPI) *SignupHandler {
	return &SignupHandler{
		dataApi:    dataApi,
		authApi:    authApi,
		addressAPI: addressAPI,
	}
}

func (s *SignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData SignupPatientRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	if !validator.IsValidEmail(requestData.Email) {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Please enter a valid email address")
		golog.Infof("Invalid email during patient signup: %s", requestData.Email)
		return
	}

	// ensure that the date of birth can be correctly parsed
	// Note that the date will be returned as MM/DD/YYYY
	dobParts := strings.Split(requestData.Dob, encoding.DOB_SEPARATOR)
	if len(dobParts) < 3 {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse dob. Format should be "+encoding.DOB_FORMAT)
		return
	}

	cityState, err := s.addressAPI.ZipcodeLookup(requestData.Zipcode)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// first, create an account for the user
	accountID, token, err := s.authApi.SignUp(requestData.Email, requestData.Password, api.PATIENT_ROLE)
	if err == api.LoginAlreadyExists {
		apiservice.WriteUserError(w, http.StatusBadRequest, "An account with the specified email address already exists.")
		return
	}

	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Internal Server Error. Unable to register patient")
		return
	}

	newPatient := &common.Patient{
		AccountId:        encoding.NewObjectId(accountID),
		FirstName:        requestData.FirstName,
		LastName:         requestData.LastName,
		Gender:           requestData.Gender,
		ZipCode:          requestData.Zipcode,
		CityFromZipCode:  cityState.City,
		StateFromZipCode: cityState.StateAbbreviation,
		PhoneNumbers: []*common.PhoneNumber{&common.PhoneNumber{
			Phone: requestData.Phone,
			Type:  api.PHONE_CELL,
		},
		},
		PromptStatus: common.Unprompted,
	}

	newPatient.Dob, err = encoding.NewDobFromComponents(dobParts[0], dobParts[1], dobParts[2])
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse date of birth. Required format + "+encoding.DOB_FORMAT)
		return
	}

	// then, register the signed up user as a patient
	err = s.dataApi.RegisterPatient(newPatient)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to register patient: "+err.Error())
		return
	}

	// track patient agreements
	if requestData.Agreements != "" {
		patientAgreements := make(map[string]bool)
		for _, agreement := range strings.Split(requestData.Agreements, ",") {
			patientAgreements[strings.TrimSpace(agreement)] = true
		}

		err = s.dataApi.TrackPatientAgreements(newPatient.PatientId.Int64(), patientAgreements)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to track patient agreements: "+err.Error())
			return
		}
	}

	// create care team for patient
	if requestData.DoctorId != 0 {
		_, err = s.dataApi.CreateCareTeamForPatientWithPrimaryDoctor(newPatient.PatientId.Int64(), apiservice.HEALTH_CONDITION_ACNE_ID, requestData.DoctorId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientSignedupResponse{Token: token, Patient: newPatient})
}
