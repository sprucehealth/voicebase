package patient

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/auth"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

var (
	acceptableWindow = 10 * time.Minute
)

type SignupHandler struct {
	dataApi            api.DataAPI
	authApi            api.AuthAPI
	analyticsLogger    analytics.Logger
	dispatcher         *dispatch.Dispatcher
	addressAPI         address.AddressValidationAPI
	store              storage.Store
	rateLimiter        ratelimit.KeyedRateLimiter
	expirationDuration time.Duration
	statAttempted      *metrics.Counter
	statSucceeded      *metrics.Counter
	statRateLimited    *metrics.Counter
}

type promotionConfirmationContent struct {
	NavBarTitle string `json:"nav_bar_title"`
	Title       string `json:"title"`
	BodyText    string `json:"body_text"`
	ButtonTitle string `json:"button_title"`
}

type PatientSignedupResponse struct {
	Token                        string                        `json:"token"`
	Patient                      *common.Patient               `json:"patient,omitempty"`
	PatientVisitData             *PatientVisitResponse         `json:"patient_visit_data,omitempty"`
	PromotionConfirmationContent *promotionConfirmationContent `json:"promotion_confirmation_content"`
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
	Training    bool   `schema:"training"`
}

type helperData struct {
	cityState    *address.CityState
	patientPhone common.Phone
	patientDOB   encoding.DOB
}

func NewSignupHandler(dataApi api.DataAPI,
	authApi api.AuthAPI,
	analyticsLogger analytics.Logger,
	dispatcher *dispatch.Dispatcher,
	expirationDuration time.Duration,
	store storage.Store,
	rateLimiter ratelimit.KeyedRateLimiter,
	addressAPI address.AddressValidationAPI,
	metricsRegistry metrics.Registry,
) *SignupHandler {
	sh := &SignupHandler{
		dataApi:            dataApi,
		authApi:            authApi,
		analyticsLogger:    analyticsLogger,
		dispatcher:         dispatcher,
		addressAPI:         addressAPI,
		store:              store,
		rateLimiter:        rateLimiter,
		expirationDuration: expirationDuration,
		statAttempted:      metrics.NewCounter(),
		statSucceeded:      metrics.NewCounter(),
		statRateLimited:    metrics.NewCounter(),
	}
	metricsRegistry.Add("attempted", sh.statAttempted)
	metricsRegistry.Add("succeeded", sh.statSucceeded)
	metricsRegistry.Add("rate-limited", sh.statRateLimited)
	return sh
}

func (s *SignupHandler) validate(requestData *SignupPatientRequestData, r *http.Request) (*helperData, error) {
	s.statAttempted.Inc(1)

	if ok, err := s.rateLimiter.Check("patient-signup:"+r.RemoteAddr, 1); err != nil {
		golog.Errorf("Rate limit check failed: %s", err.Error())
	} else if !ok {
		s.statRateLimited.Inc(1)
		return nil, apiservice.NewAccessForbiddenError()
	}

	if !email.IsValidEmail(requestData.Email) {
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
	// if there is no stateCode provided by the client, use the addressAPI
	// to resolve the zipcode to state
	if requestData.StateCode == "" {
		data.cityState, err = s.addressAPI.ZipcodeLookup(requestData.Zipcode)
		if err == address.InvalidZipcodeError {
			return nil, apiservice.NewValidationError("Enter a valid zipcode", r)
		} else if err != nil {
			return nil, err
		}
	} else {
		state, err := s.dataApi.GetFullNameForState(requestData.StateCode)
		if err == api.NoRowsError {
			return nil, apiservice.NewValidationError("Invalid state code", r)
		} else if err != nil {
			return nil, err
		}

		data.cityState = &address.CityState{
			State:             state,
			StateAbbreviation: requestData.StateCode,
		}
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
		// if the account already exits, treat the signup as an update if the login credentials match
		// and we're still within an acceptable window of the registration date
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
		Training:         requestData.Training,
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
		_, err = s.dataApi.CreateCareTeamForPatientWithPrimaryDoctor(newPatient.PatientId.Int64(), api.HEALTH_CONDITION_ACNE_ID, requestData.DoctorId)
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
		pvData, err = createPatientVisit(newPatient, s.dataApi, s.dispatcher, s.store, s.expirationDuration, r, nil)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	var promoContent *promotionConfirmationContent
	successMsg, err := promotions.PatientSignedup(newPatient.AccountId.Int64(), requestData.Email, s.dataApi, s.analyticsLogger)
	if err != nil {
		golog.Errorf(err.Error())
	} else if successMsg != "" {
		promoContent = &promotionConfirmationContent{
			NavBarTitle: "Account Created",
			Title:       fmt.Sprintf("Welcome to Spruce, %s.", newPatient.FirstName),
			BodyText:    successMsg,
			ButtonTitle: "Continue",
		}
	}

	headers := apiservice.ExtractSpruceHeaders(r)
	s.dispatcher.PublishAsync(&auth.AuthenticatedEvent{
		AccountID:     newPatient.AccountId.Int64(),
		SpruceHeaders: headers,
	})

	s.statSucceeded.Inc(1)

	apiservice.WriteJSON(w, PatientSignedupResponse{
		Token:                        token,
		Patient:                      newPatient,
		PatientVisitData:             pvData,
		PromotionConfirmationContent: promoContent,
	})
}
