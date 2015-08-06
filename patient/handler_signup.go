package patient

import (
	"encoding/json"
	"errors"
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
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/media"
)

var (
	acceptableWindow = 10 * time.Minute
)

// SignupHandler represents the data associated with the handler that will process patient signup requests
type SignupHandler struct {
	dataAPI            api.DataAPI
	authAPI            api.AuthAPI
	apiDomain          string
	webDomain          string
	analyticsLogger    analytics.Logger
	dispatcher         *dispatch.Dispatcher
	addressAPI         address.Validator
	mediaStore         *media.Store
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

// SignedupResponse represents the data returned by a successful POST request
type SignedupResponse struct {
	Token                        string                        `json:"token"`
	Patient                      *common.Patient               `json:"patient,omitempty"`
	PatientVisitData             *PatientVisitResponse         `json:"patient_visit_data,omitempty"`
	PromotionConfirmationContent *promotionConfirmationContent `json:"promotion_confirmation_content"`
}

// SignupPatientRequestData represents the data associated with a sucessful POST request
type SignupPatientRequestData struct {
	Email               string `schema:"email,required" json:"email"`
	Password            string `schema:"password,required" json:"password"`
	FirstName           string `schema:"first_name,required" json:"first_name"`
	LastName            string `schema:"last_name,required" json:"last_name"`
	DOB                 string `schema:"dob,required" json:"dob"`
	Gender              string `schema:"gender,required" json:"gender"`
	ZipCode             string `schema:"zip_code,required" json:"zip_code"`
	Phone               string `schema:"phone" json:"phone"`
	Agreements          string `schema:"agreements" json:"agreements"`
	DoctorID            int64  `schema:"care_provider_id" json:"care_provider_id,string"`
	StateCode           string `schema:"state_code" json:"state_code"`
	CreateVisit         bool   `schema:"create_visit" json:"create_visit"`
	Training            bool   `schema:"training" json:"training"`
	PathwayTag          string `schema:"pathway_id" json:"pathway_id"`
	AttributionDataJSON string `schema:"attribution_data" json:"attribution_data"`
}

type helperData struct {
	cityState    *address.CityState
	patientPhone common.Phone
	patientDOB   encoding.Date
}

type attributionData struct {
	PromoCode string `json:"promo_code"`
}

// NewSignupHandler returns and initialized instance of SignupHandler
func NewSignupHandler(
	dataAPI api.DataAPI,
	authAPI api.AuthAPI,
	apiDomain string,
	webDomain string,
	analyticsLogger analytics.Logger,
	dispatcher *dispatch.Dispatcher,
	expirationDuration time.Duration,
	mediaStore *media.Store,
	rateLimiter ratelimit.KeyedRateLimiter,
	addressAPI address.Validator,
	metricsRegistry metrics.Registry,
) http.Handler {
	sh := &SignupHandler{
		dataAPI:            dataAPI,
		authAPI:            authAPI,
		apiDomain:          apiDomain,
		webDomain:          webDomain,
		analyticsLogger:    analyticsLogger,
		dispatcher:         dispatcher,
		addressAPI:         addressAPI,
		mediaStore:         mediaStore,
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
		httputil.Post)
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
	dobParts := strings.Split(requestData.DOB, encoding.DateSeparator)
	if len(dobParts) < 3 {
		return nil, apiservice.NewValidationError("Unable to parse dob. Format should be " + encoding.DateFormat)
	}

	data := &helperData{}
	var err error
	// if there is no stateCode provided by the client, use the addressAPI
	// to resolve the zipcode to state
	if requestData.StateCode == "" {
		data.cityState, err = s.addressAPI.ZipcodeLookup(requestData.ZipCode)
		if err == address.ErrInvalidZipcode {
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

	data.patientDOB, err = encoding.NewDateFromComponents(dobParts[0], dobParts[1], dobParts[2])
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

	requestData.Email = strings.TrimSpace(strings.ToLower(requestData.Email))

	data, err := s.validate(&requestData, r)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// first, create an account for the user
	var update bool
	var patientID int64
	accountID, err := s.authAPI.CreateAccount(requestData.Email, requestData.Password, api.RolePatient)
	if err == api.ErrLoginAlreadyExists {
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
				Type:  common.PNTCell,
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
		newPatient.ID = encoding.NewObjectID(patientID)
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

		err = s.dataAPI.TrackPatientAgreements(newPatient.ID.Int64(), patientAgreements)
		if err != nil {
			apiservice.WriteError(errors.New("Unable to track patient agreements: "+err.Error()), w, r)
			return
		}
	}

	if requestData.PathwayTag == "" {
		// by default assume acne for backwards compatibility
		requestData.PathwayTag = api.AcnePathwayTag
	}

	token, err := s.authAPI.CreateToken(accountID, api.Mobile, 0)
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
			s.webDomain,
			s.dispatcher,
			s.mediaStore,
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

	if requestData.AttributionDataJSON != "" {
		// To provide a good account creation experience don't block on promo code association/metric emission
		conc.Go(func() { s.applyAttribution(r, accountID, newPatient, requestData) })
	}

	headers := apiservice.ExtractSpruceHeaders(r)
	s.dispatcher.PublishAsync(&SignupEvent{
		AccountID:     newPatient.AccountID.Int64(),
		PatientID:     newPatient.ID.Int64(),
		SpruceHeaders: headers,
	})
	s.dispatcher.PublishAsync(&auth.AuthenticatedEvent{
		AccountID:     newPatient.AccountID.Int64(),
		SpruceHeaders: headers,
	})

	s.statSucceeded.Inc(1)

	httputil.JSONResponse(w, http.StatusOK, SignedupResponse{
		Token:                        token,
		Patient:                      newPatient,
		PatientVisitData:             pvData,
		PromotionConfirmationContent: promoContent,
	})
}

func (s *SignupHandler) applyAttribution(r *http.Request, accountID int64, newPatient *common.Patient, requestData SignupPatientRequestData) {
	var attrData attributionData
	if err := json.Unmarshal([]byte(requestData.AttributionDataJSON), &attrData); err != nil {
		golog.Errorf(err.Error())
		return
	}

	// Also dump everything that came along into a map to stick onto metrics
	var attributionMap map[string]interface{}
	if err := json.Unmarshal([]byte(requestData.AttributionDataJSON), &attributionMap); err != nil {
		golog.Errorf(err.Error())
		return
	}
	if attrData.PromoCode != "" {
		promoCode, err := s.dataAPI.LookupPromoCode(attrData.PromoCode)
		if api.IsErrNotFound(err) {
			golog.Warningf("Promotion code in attribution data could not be found %s", attrData.PromoCode)
			return
		} else if err != nil {
			golog.Errorf("Unable to lookup attribution promo code %s at account creation time: %v", attrData.PromoCode, err)
			return
		}
		code := promoCode.Code
		codeID := promoCode.ID
		if promoCode.IsReferral {
			rp, err := s.dataAPI.ReferralProgram(promoCode.ID, common.PromotionTypes)
			if err != nil {
				golog.Errorf(err.Error())
				return
			}
			if err := rp.Data.(promotions.ReferralProgram).ReferredAccountAssociatedCode(accountID, promoCode.ID, s.dataAPI); err != nil {
				golog.Errorf(err.Error())
				return
			}
			if rp.TemplateID != nil {
				rpt, err := s.dataAPI.ReferralProgramTemplate(*rp.TemplateID, common.PromotionTypes)
				if err != nil {
					golog.Errorf(err.Error())
					return
				}
				if rpt.PromotionCodeID != nil {
					promotion, err := s.dataAPI.Promotion(*rpt.PromotionCodeID, common.PromotionTypes)
					if err != nil {
						golog.Errorf(err.Error())
						return
					}
					code = promotion.Code
					codeID = promotion.CodeID
				}
			}
			s.analyticsLogger.WriteEvents([]analytics.Event{
				&analytics.ServerEvent{
					Event:     "referral_code_account_created",
					Timestamp: analytics.Time(time.Now()),
					AccountID: rp.AccountID,
					ExtraJSON: analytics.JSONString(struct {
						CreatedAccountID int64                  `json:"created_account_id"`
						Code             string                 `json:"code"`
						CodeID           int64                  `json:"code_id"`
						AttributionData  map[string]interface{} `json:"attribution_data"`
					}{
						CreatedAccountID: accountID,
						Code:             code,
						CodeID:           codeID,
						AttributionData:  attributionMap,
					}),
				},
			})
		}

		s.analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "promo_code_account_created",
				Timestamp: analytics.Time(time.Now()),
				AccountID: accountID,
				ExtraJSON: analytics.JSONString(struct {
					Code            string                 `json:"code"`
					CodeID          int64                  `json:"code_id"`
					AttributionData map[string]interface{} `json:"attribution_data"`
				}{
					Code:            code,
					CodeID:          promoCode.ID,
					AttributionData: attributionMap,
				}),
			},
		})

		async := false
		_, err = promotions.AssociatePromoCode(newPatient.Email, newPatient.StateFromZipCode, attrData.PromoCode, s.dataAPI, s.authAPI, s.analyticsLogger, async)
		if err != nil {
			golog.Errorf("Unable associate promo code %s at account creation time: %v", attrData.PromoCode, err)
		}
	}
}
