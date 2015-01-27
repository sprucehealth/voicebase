package patient

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
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
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/storage"
)

var (
	acceptableWindow = 10 * time.Minute
)

type SignupHandler struct {
	dataAPI            api.DataAPI
	authAPI            api.AuthAPI
	apiDomain          string
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

type SignupPatientRequestData struct {
	Email       string `schema:"email,required" json:"email"`
	Password    string `schema:"password,required" json:"password"`
	FirstName   string `schema:"first_name,required" json:"first_name"`
	LastName    string `schema:"last_name,required" json:"last_name"`
	DOB         string `schema:"dob,required" json:"dob"`
	Gender      string `schema:"gender,required" json:"gender"`
	ZipCode     string `schema:"zip_code,required" json:"zip_code"`
	Phone       string `schema:"phone" json:"phone"`
	Agreements  string `schema:"agreements" json:"agreements"`
	DoctorID    int64  `schema:"care_provider_id" json:"doctor_id,string"`
	StateCode   string `schema:"state_code" json:"state_code"`
	CreateVisit bool   `schema:"create_visit" json:"create_visit"`
	Training    bool   `schema:"training" json:"training"`
	PathwayTag  string `schema:"pathway_id" json:"pathway_id"`
}

type helperData struct {
	cityState    *address.CityState
	patientPhone common.Phone
	patientDOB   encoding.DOB
}

func NewSignupHandler(
	dataAPI api.DataAPI,
	authAPI api.AuthAPI,
	apiDomain string,
	analyticsLogger analytics.Logger,
	dispatcher *dispatch.Dispatcher,
	expirationDuration time.Duration,
	store storage.Store,
	rateLimiter ratelimit.KeyedRateLimiter,
	addressAPI address.AddressValidationAPI,
	metricsRegistry metrics.Registry,
) http.Handler {
	sh := &SignupHandler{
		dataAPI:            dataAPI,
		authAPI:            authAPI,
		apiDomain:          apiDomain,
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
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(sh),
		[]string{"POST"})
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
		return nil, apiservice.NewValidationError("Please enter a valid email address")
	}

	// ensure that the date of birth can be correctly parsed
	// Note that the date will be returned as MM/DD/YYYY
	dobParts := strings.Split(requestData.DOB, encoding.DOBSeparator)
	if len(dobParts) < 3 {
		return nil, apiservice.NewValidationError("Unable to parse dob. Format should be " + encoding.DOBFormat)
	}

	data := &helperData{}
	var err error
	// if there is no stateCode provided by the client, use the addressAPI
	// to resolve the zipcode to state
	if requestData.StateCode == "" {
		data.cityState, err = s.addressAPI.ZipcodeLookup(requestData.ZipCode)
		if err == address.InvalidZipcodeError {
			return nil, apiservice.NewValidationError("Enter a valid zip code")
		} else if err != nil {
			return nil, err
		}
	} else {
		state, _, err := s.dataAPI.State(requestData.StateCode)
		if api.IsErrNotFound(err) {
			return nil, apiservice.NewValidationError("Invalid state code")
		} else if err != nil {
			return nil, err
		}

		data.cityState = &address.CityState{
			State:             state,
			StateAbbreviation: requestData.StateCode,
		}
	}

	if requestData.Phone != "" {
		data.patientPhone, err = common.ParsePhone(requestData.Phone)
		if err != nil {
			return nil, apiservice.NewValidationError(err.Error())
		}
	}

	data.patientDOB, err = encoding.NewDOBFromComponents(dobParts[0], dobParts[1], dobParts[2])
	if err != nil {
		return nil, apiservice.NewValidationError(err.Error())
	}
	return data, nil
}

func (s *SignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	var requestData SignupPatientRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
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
	accountID, err := s.authAPI.CreateAccount(requestData.Email, requestData.Password, api.PATIENT_ROLE)
	if err == api.LoginAlreadyExists {
		// if the account already exits, treat the signup as an update if the login credentials match
		// and we're still within an acceptable window of the registration date
		account, err := s.authAPI.Authenticate(requestData.Email, requestData.Password)
		if err != nil {
			apiservice.WriteValidationError("An account with the specified email address already exists.", w, r)
			return
		} else if account.Registered.Add(acceptableWindow).Before(time.Now()) {
			apiservice.WriteValidationError("An account with the specified email address already exists.", w, r)
			return
		}

		update = true
		accountID = account.ID
		patientID, err = s.dataAPI.GetPatientIDFromAccountID(accountID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	newPatient := &common.Patient{
		AccountID:        encoding.NewObjectID(accountID),
		Email:            requestData.Email,
		FirstName:        requestData.FirstName,
		LastName:         requestData.LastName,
		Gender:           requestData.Gender,
		ZipCode:          requestData.ZipCode,
		CityFromZipCode:  data.cityState.City,
		StateFromZipCode: data.cityState.StateAbbreviation,
		PromptStatus:     common.Unprompted,
		DOB:              data.patientDOB,
		Training:         requestData.Training,
	}

	if data.patientPhone.String() != "" {
		newPatient.PhoneNumbers = append(newPatient.PhoneNumbers,
			&common.PhoneNumber{
				Phone: data.patientPhone,
				Type:  api.PHONE_CELL,
			})
	}

	if update {
		patientUpdate := &api.PatientUpdate{
			FirstName:    &requestData.FirstName,
			LastName:     &requestData.LastName,
			DOB:          &data.patientDOB,
			Gender:       &requestData.Gender,
			PhoneNumbers: newPatient.PhoneNumbers,
		}
		if err := s.dataAPI.UpdatePatient(patientID, patientUpdate, false); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		newPatient.PatientID = encoding.NewObjectID(patientID)
	} else {
		// then, register the signed up user as a patient
		if err := s.dataAPI.RegisterPatient(newPatient); err != nil {
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

		err = s.dataAPI.TrackPatientAgreements(newPatient.PatientID.Int64(), patientAgreements)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to track patient agreements: "+err.Error())
			return
		}
	}

	if requestData.PathwayTag == "" {
		// by default assume acne for backwards compatibility
		requestData.PathwayTag = api.AcnePathwayTag
	}

	token, err := s.authAPI.CreateToken(accountID, api.Mobile, api.RegularAuth)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var pvData *PatientVisitResponse
	if requestData.CreateVisit {
		var err error
		pvData, err = createPatientVisit(
			newPatient,
			requestData.DoctorID,
			requestData.PathwayTag,
			s.dataAPI,
			s.apiDomain,
			s.dispatcher,
			s.store,
			s.expirationDuration,
			r, nil)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	var promoContent *promotionConfirmationContent
	successMsg, err := promotions.PatientSignedup(newPatient.AccountID.Int64(), requestData.Email, s.dataAPI, s.analyticsLogger)
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
		AccountID:     newPatient.AccountID.Int64(),
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
