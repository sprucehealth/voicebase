package patient

import (
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/storage"

	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/dchest/validator"
)

var (
	acceptableWindow = 20 * time.Minute
)

type SignupHandler struct {
	dataApi            api.DataAPI
	authApi            api.AuthAPI
	addressAPI         address.AddressValidationAPI
	store              storage.Store
	expirationDuration time.Duration
}

type PatientSignedupResponse struct {
	Token            string                `json:"token"`
	Patient          *common.Patient       `json:"patient,omitempty"`
	PatientVisitData *PatientVisitResponse `json:"patient_visit_data,omitempty"`
}

func (s *SignupHandler) NonAuthenticated() bool {
	return true
}

func (s *SignupHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

type SignupPatientRequestData struct {
	Email       string `schema:"email,required"`
	Password    string `schema:"password,required"`
	FirstName   string `schema:"first_name,required"`
	LastName    string `schema:"last_name,required"`
	DOB         string `schema:"dob,required"`
	Gender      string `schema:"gender,required"`
	Zipcode     string `schema:"zip_code,required"`
	Phone       string `schema:"phone,required"`
	Agreements  string `schema:"agreements"`
	DoctorId    int64  `schema:"doctor_id"`
	StateCode   string `schema:"state_code"`
	CreateVisit bool   `schema:"create_visit"`
}

type helperData struct {
	cityState    *address.CityState
	patientPhone common.Phone
	patientDOB   encoding.DOB
}

func NewSignupHandler(dataApi api.DataAPI,
	authApi api.AuthAPI,
	expirationDuration time.Duration,
	store storage.Store,
	addressAPI address.AddressValidationAPI) *SignupHandler {
	return &SignupHandler{
		dataApi:            dataApi,
		authApi:            authApi,
		addressAPI:         addressAPI,
		store:              store,
		expirationDuration: expirationDuration,
	}
}

func (s *SignupHandler) validate(requestData *SignupPatientRequestData, r *http.Request) (*helperData, error) {
	if !validator.IsValidEmail(requestData.Email) {
		return nil, apiservice.NewValidationError("Please enter a valid email address", r)
	}

	// ensure that the date of birth can be correctly parsed
	// Note that the date will be returned as MM/DD/YYYY
	dobParts := strings.Split(requestData.DOB, encoding.DOBSeparator)
	if len(dobParts) < 3 {
		return nil, apiservice.NewValidationError("Unable to parse dob. Format should be "+encoding.DOBFormat, r)
	}

	data := &helperData{}
	var err error
	data.cityState, err = s.addressAPI.ZipcodeLookup(requestData.Zipcode)
	if err != nil {
		return nil, err
	}

	data.patientPhone, err = common.ParsePhone(requestData.Phone)
	if err != nil {
		return nil, apiservice.NewValidationError(err.Error(), r)
	}

	data.patientDOB, err = encoding.NewDOBFromComponents(dobParts[0], dobParts[1], dobParts[2])
	if err != nil {
		return nil, apiservice.NewValidationError(err.Error(), r)
	}
	return data, nil
}

func (s *SignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	var requestData SignupPatientRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	data, err := s.validate(&requestData, r)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// first, create an account for the user
	var update bool
	var patientID int64
	accountID, err := s.authApi.CreateAccount(requestData.Email, requestData.Password, api.PATIENT_ROLE)
	if err == api.LoginAlreadyExists {
		// if the account already exits, it's possible that the client did not get the response to register that
		// the account was created. If created within the acceptable window, treat the account signup as an update
		// to ensure that we capture all the relevant data from the client
		account, err := s.authApi.Authenticate(requestData.Email, requestData.Password)
		if err != nil {
			apiservice.WriteValidationError("An account with the specified email address already exists.", w, r)
			return
		} else if account.Registered.Add(acceptableWindow).Before(time.Now()) {
			apiservice.WriteValidationError("An account with the specified email address already exists.", w, r)
			return
		}

		update = true
		accountID = account.ID
		patientID, err = s.dataApi.GetPatientIdFromAccountId(accountID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	newPatient := &common.Patient{
		AccountId:        encoding.NewObjectId(accountID),
		Email:            requestData.Email,
		FirstName:        requestData.FirstName,
		LastName:         requestData.LastName,
		Gender:           requestData.Gender,
		ZipCode:          requestData.Zipcode,
		CityFromZipCode:  data.cityState.City,
		StateFromZipCode: data.cityState.StateAbbreviation,
		PromptStatus:     common.Unprompted,
		DOB:              data.patientDOB,
		PhoneNumbers: []*common.PhoneNumber{&common.PhoneNumber{
			Phone: data.patientPhone,
			Type:  api.PHONE_CELL,
		},
		},
	}

	if update {
		newPatient.PatientId = encoding.NewObjectId(patientID)
		if err := s.dataApi.UpdateTopLevelPatientInformation(newPatient); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	} else {
		// then, register the signed up user as a patient
		if err := s.dataApi.RegisterPatient(newPatient); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
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

	token, err := s.authApi.CreateToken(accountID, api.Mobile, api.RegularAuth)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var pvData *PatientVisitResponse
	if requestData.CreateVisit {
		var err error
		pvData, err = createPatientVisit(newPatient, s.dataApi, s.store, s.expirationDuration, r)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSON(w, PatientSignedupResponse{
		Token:            token,
		Patient:          newPatient,
		PatientVisitData: pvData,
	})
}
