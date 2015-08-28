package router

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/rainycape/memcache"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/appevent"
	"github.com/sprucehealth/backend/auth"
	"github.com/sprucehealth/backend/careprovider"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/compat"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/demo"
	"github.com/sprucehealth/backend/diagnosis"
	diaghandlers "github.com/sprucehealth/backend/diagnosis/handlers"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/email/campaigns"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/features"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/medrecord"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/misc/handlers"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/passreset"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_case"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/reslib"
	"github.com/sprucehealth/backend/settings"
	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/treatment_plan"
	"golang.org/x/net/context"
)

// Config is all services and configuration values used by the app api handlers.
type Config struct {
	DataAPI                  api.DataAPI
	AuthAPI                  api.AuthAPI
	Dispatcher               *dispatch.Dispatcher
	AuthTokenExpiration      time.Duration
	MediaAccessExpiration    time.Duration
	AddressValidator         address.Validator
	PharmacySearchAPI        pharmacy.PharmacySearchAPI
	DiagnosisAPI             diagnosis.API
	SNSClient                snsiface.SNSAPI
	PaymentAPI               apiservice.StripeClient
	MemcacheClient           *memcache.Client
	NotifyConfigs            *config.NotificationConfigs
	MinimumAppVersionConfigs *config.MinimumAppVersionConfigs
	DosespotConfig           *config.DosespotConfig
	NotificationManager      *notify.NotificationManager
	ERxStatusQueue           *common.SQSQueue
	ERxRoutingQueue          *common.SQSQueue
	ERxAPI                   erx.ERxAPI
	MedicalRecordQueue       *common.SQSQueue
	VisitQueue               *common.SQSQueue
	EmailService             email.Service
	MetricsRegistry          metrics.Registry
	SMSAPI                   api.SMSAPI
	Stores                   storage.StoreMap
	MediaStore               *media.Store
	RateLimiters             ratelimit.KeyedRateLimiters
	AnalyticsLogger          analytics.Logger
	ERxRouting               bool
	LaunchPromoStartDate     *time.Time
	JBCQMinutesThreshold     int
	NumDoctorSelection       int
	CustomerSupportEmail     string
	TechnicalSupportEmail    string
	APIDomain                string
	WebDomain                string
	APICDNDomain             string
	StaticContentURL         string
	StaticResourceURL        string
	AWSRegion                string
	TwoFactorExpiration      int
	SMSFromNumber            string
	Cfg                      cfg.Store
	ApplicationDB            *sql.DB
	Signer                   *sig.Signer

	mux *mux.Router
}

// New returns an initialized instance of the apiservice router conforming to the http.Handler interface
func New(conf *Config) (*mux.Router, httputil.ContextHandler) {
	taggingClient := tagging.NewTaggingClient(conf.ApplicationDB)

	// Initialize listneners
	doctor_queue.InitListeners(conf.DataAPI, conf.AnalyticsLogger, conf.Dispatcher, conf.NotificationManager, conf.MetricsRegistry.Scope("doctor_queue"), conf.JBCQMinutesThreshold, conf.CustomerSupportEmail, conf.Cfg, taggingClient)
	doctor_treatment_plan.InitListeners(conf.DataAPI, conf.Dispatcher)
	notify.InitListeners(conf.DataAPI, conf.Dispatcher)
	patient_case.InitListeners(conf.DataAPI, conf.Dispatcher, conf.NotificationManager)
	demo.InitListeners(conf.DataAPI, conf.Dispatcher, conf.APIDomain, conf.DosespotConfig.UserID)
	patient_visit.InitListeners(conf.DataAPI, conf.APICDNDomain, conf.Dispatcher, conf.VisitQueue, taggingClient)
	doctor.InitListeners(conf.DataAPI, conf.APICDNDomain, conf.Dispatcher, taggingClient)
	cost.InitListeners(conf.DataAPI, conf.Dispatcher)
	auth.InitListeners(conf.AuthAPI, conf.Dispatcher)
	campaigns.InitListeners(conf.Dispatcher, conf.Cfg, conf.EmailService, conf.DataAPI, conf.WebDomain)
	messages.InitListeners(conf.DataAPI, conf.Dispatcher)

	// TODO: for now hardcoding this here until I can figure out the best place to store it.
	// Possible config, possibly cfg, possible something else
	var appFeatures compat.Features
	appFeatures.Register([]*compat.Feature{
		{
			Name: features.MsgAttachGuide,
			AppVersions: map[string]encoding.VersionRange{
				"ios-patient": {MinVersion: &encoding.Version{2, 1, 0}},
			},
		},
		{
			Name: features.OldRAFHomeCard,
			AppVersions: map[string]encoding.VersionRange{
				"ios-patient": {MinVersion: &encoding.Version{1, 1, 0}, MaxVersion: &encoding.Version{2, 0, 2}},
			},
		},
		{
			Name: features.RAFHomeCard,
			AppVersions: map[string]encoding.VersionRange{
				"ios-patient": {MinVersion: &encoding.Version{2, 0, 2}},
			},
		},
	})

	conf.mux = mux.NewRouter()

	addressValidationAPI := address.NewAddressValidationWithCacheWrapper(conf.AddressValidator, conf.MemcacheClient)

	// Patient/Doctor: Push notification APIs
	authenticationRequired(conf, apipaths.NotificationTokenURLPath, notify.NewNotificationHandler(conf.DataAPI, conf.NotifyConfigs, conf.SNSClient))
	authenticationRequired(conf, apipaths.NotificationPromptStatusURLPath, notify.NewPromptStatusHandler(conf.DataAPI))

	// Patient: Account related APIs

	noAuthenticationRequired(conf, apipaths.AuthCheckEmailURLPath,
		auth.NewCheckEmailHandler(conf.AuthAPI, conf.RateLimiters.Get("check_email"),
			conf.MetricsRegistry.Scope("auth.check_email")))
	authenticationRequired(conf, apipaths.PatientUpdateURLPath, patient.NewUpdateHandler(conf.DataAPI, conf.AddressValidator))
	authenticationRequired(conf, apipaths.PatientPharmacyURLPath, patient.NewPharmacyHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientAlertsURLPath, patient_file.NewAlertsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientIsAuthenticatedURLPath, handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	authenticationRequired(conf, apipaths.PatientCardURLPath, patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidator))
	authenticationRequired(conf, apipaths.PatientReplaceCardURLPath, patient.NewReplaceCardHandler(conf.DataAPI, conf.PaymentAPI))
	authenticationRequired(conf, apipaths.PatientDefaultCardURLPath, patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidator))
	authenticationRequired(conf, apipaths.PatientRequestMedicalRecordURLPath, medrecord.NewRequestAPIHandler(conf.DataAPI, conf.MedicalRecordQueue))
	authenticationRequired(conf, apipaths.LogoutURLPath, patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.Dispatcher, conf.StaticContentURL,
		conf.RateLimiters.Get("login"), conf.MetricsRegistry.Scope("patient.auth")))
	authenticationRequired(conf, apipaths.PatientPCPURLPath, patient.NewPCPHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientEmergencyContactsURLPath, patient.NewEmergencyContactsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientMeURLPath, patient.NewMeHandler(conf.DataAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.PatientCareTeamURLPath, patient.NewCareTeamHandler(conf.DataAPI, conf.APICDNDomain))
	authenticationRequired(conf, apipaths.PatientCareTeamsURLPath, patient_file.NewPatientCareTeamsHandler(conf.DataAPI, conf.APICDNDomain))
	authenticationRequired(conf, apipaths.PatientCostURLPath, cost.NewCostHandler(conf.DataAPI, conf.AnalyticsLogger, conf.Cfg))
	authenticationRequired(conf, apipaths.PatientCreditsURLPath, promotions.NewPatientCreditsHandler(conf.DataAPI))
	noAuthenticationRequired(conf, apipaths.PatientSignupURLPath, patient.NewSignupHandler(
		conf.DataAPI, conf.AuthAPI, conf.APICDNDomain, conf.WebDomain, conf.AnalyticsLogger, conf.Dispatcher, conf.AuthTokenExpiration,
		conf.MediaStore, conf.RateLimiters.Get("patient-signup"), addressValidationAPI,
		conf.MetricsRegistry.Scope("patient.signup")))
	noAuthenticationRequired(conf, apipaths.PatientAuthenticateURLPath, patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.Dispatcher,
		conf.StaticContentURL, conf.RateLimiters.Get("login"), conf.MetricsRegistry.Scope("patient.auth")))

	// Patient: Patient Case Related APIs
	authenticationRequired(conf, apipaths.PatientVisitURLPath, patient.NewPatientVisitHandler(
		conf.DataAPI,
		conf.AuthAPI,
		conf.PaymentAPI,
		conf.AddressValidator,
		conf.APICDNDomain,
		conf.WebDomain,
		conf.Dispatcher,
		conf.MediaStore,
		conf.AuthTokenExpiration,
		taggingClient))
	authenticationRequired(conf, apipaths.PatientVisitsListURLPath, patient.NewVisitsListHandler(conf.DataAPI, conf.APICDNDomain, conf.WebDomain, conf.Dispatcher, conf.MediaStore, conf.AuthTokenExpiration))
	authenticationRequired(conf, apipaths.PatientVisitIntakeURLPath, patient_visit.NewAnswerIntakeHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientVisitTriageURLPath, patient_visit.NewPreSubmissionTriageHandler(conf.DataAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.PatientVisitMessageURLPath, patient_visit.NewMessageHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientVisitPhotoAnswerURLPath, patient_visit.NewPhotoAnswerIntakeHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientVisitReachedConsentStep, patient_visit.NewReachedConsentStep(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientTreatmentsURLPath, treatment_plan.NewTreatmentsHandler(conf.DataAPI))
	noAuthenticationRequired(conf, apipaths.CheckEligibilityURLPath, patient.NewCheckCareProvidingEligibilityHandler(conf.DataAPI, addressValidationAPI, conf.AnalyticsLogger))

	authenticationRequired(conf, apipaths.TreatmentPlanURLPath, treatment_plan.NewTreatmentPlanHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.TreatmentGuideURLPath, treatment_plan.NewTreatmentGuideHandler(conf.DataAPI))
	noAuthenticationRequired(conf, apipaths.RXGuideURLPath, treatment_plan.NewRXGuideHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.AutocompleteURLPath, handlers.NewAutocompleteHandler(conf.DataAPI, conf.ERxAPI))
	authenticationRequired(conf, apipaths.PharmacySearchURLPath, patient.NewPharmacySearchHandler(conf.DataAPI, conf.PharmacySearchAPI))

	// Patient: Home APIs
	noAuthenticationRequired(conf, apipaths.PatientHomeURLPath, patient_case.NewHomeHandler(conf.DataAPI, conf.APICDNDomain, conf.WebDomain, addressValidationAPI))
	noAuthenticationRequired(conf, apipaths.PatientHowFAQURLPath, handlers.NewPatientFAQHandler(conf.StaticContentURL))
	noAuthenticationRequired(conf, apipaths.PatientPricingFAQURLPath, handlers.NewPricingFAQHandler(conf.StaticContentURL))
	noAuthenticationRequired(conf, apipaths.PatientFeaturedDoctorsURLPath, handlers.NewFeaturedDoctorsHandler(conf.StaticContentURL))

	// Patient/Doctor: Case APIs
	authenticationRequired(conf, apipaths.PatientCasesURLPath, patient_case.NewCaseInfoHandler(conf.DataAPI, conf.APICDNDomain))

	// Doctor: Case APIs
	authenticationRequired(conf, apipaths.CaseNotesURLPath, patient_case.NewPatientCaseNoteHandler(conf.DataAPI, conf.APIDomain))
	authenticationRequired(conf, apipaths.CasePatientFeedbackURLPath, patient_case.NewPatientFeedbackHandler(conf.DataAPI))

	// Patient: Case APIs
	authenticationRequired(conf, apipaths.PatientCaseNotificationsURLPath, patient_case.NewNotificationsListHandler(conf.DataAPI, conf.APICDNDomain))

	// Patient/Doctor: Resource guide APIs
	noAuthenticationRequired(conf, apipaths.ResourceGuideURLPath, reslib.NewHandler(conf.DataAPI))
	noAuthenticationRequired(conf, apipaths.ResourceGuidesListURLPath, reslib.NewListHandler(conf.DataAPI))

	// Patient/Doctor: Message APIs
	authenticationRequired(conf, apipaths.CaseMessagesURLPath, messages.NewHandler(conf.DataAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.CaseMessagesListURLPath, messages.NewListHandler(conf.DataAPI, conf.APICDNDomain, conf.MediaStore, conf.AuthTokenExpiration))
	authenticationRequired(conf, apipaths.CaseMessagesUnreadCountURLPath, messages.NewUnreadCountHandler(conf.DataAPI))

	// Doctor: Account APIs
	authenticationRequired(conf, apipaths.DoctorIsAuthenticatedURLPath, handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	authenticationRequired(conf, apipaths.DoctorQueueURLPath, doctor_queue.NewQueueHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorQueueItemURLPath, doctor_queue.NewItemHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorQueueInboxURLPath, doctor_queue.NewInboxHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorQueueUnassignedURLPath, doctor_queue.NewUnassignedHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorQueueHistoryURLPath, doctor_queue.NewHistoryHandler(conf.DataAPI))

	authenticationRequired(conf, apipaths.DoctorCaseHistoryURLPath, doctor_queue.NewPatientsFeedHandler(conf.DataAPI, taggingClient))
	noAuthenticationRequired(conf, apipaths.DoctorSignupURLPath, doctor.NewSignupDoctorHandler(conf.DataAPI, conf.AuthAPI))
	noAuthenticationRequired(conf, apipaths.DoctorAuthenticateURLPath, doctor.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.SMSAPI, conf.APICDNDomain, conf.Dispatcher,
		conf.SMSFromNumber, conf.TwoFactorExpiration, conf.RateLimiters.Get("login"), conf.MetricsRegistry.Scope("doctor.auth")))
	noAuthenticationRequired(conf, apipaths.DoctorAuthenticateTwoFactorURLPath, doctor.NewTwoFactorHandler(conf.DataAPI, conf.AuthAPI, conf.SMSAPI, conf.APICDNDomain, conf.SMSFromNumber, conf.TwoFactorExpiration))

	// Doctor: Prescription related APIs
	authenticationRequired(conf, apipaths.DoctorRXErrorURLPath, doctor.NewPrescriptionErrorHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorRXErrorResolveURLPath, doctor.NewPrescriptionErrorIgnoreHandler(conf.DataAPI, conf.ERxAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.DoctorRefillRxURLPath, doctor.NewRefillRxHandler(conf.DataAPI, conf.ERxAPI, conf.Dispatcher, conf.ERxStatusQueue))
	authenticationRequired(conf, apipaths.DoctorRefillRxDenialReasonsURLPath, doctor.NewRefillRxDenialReasonsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorFTPURLPath, doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(conf.DataAPI, conf.ERxAPI, conf.MediaStore))
	authenticationRequired(conf, apipaths.DoctorTreatmentTemplatesURLPath, doctor_treatment_plan.NewTreatmentTemplatesHandler(conf.DataAPI))

	// Doctor: Patient file APIs
	authenticationRequired(conf, apipaths.DoctorPatientTreatmentsURLPath, patient_file.NewDoctorPatientTreatmentsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorPatientInfoURLPath, patient_file.NewDoctorPatientHandler(conf.DataAPI, conf.AddressValidator))
	authenticationRequired(conf, apipaths.DoctorPatientParentInfoURLPath, patient_file.NewPatientParentHandler(conf.DataAPI, conf.MediaStore, conf.MediaAccessExpiration))
	authenticationRequired(conf, apipaths.DoctorPatientAppInfoURLPath, patient_file.NewPatientAppInfoHandler(conf.DataAPI, conf.AuthAPI))
	authenticationRequired(conf, apipaths.DoctorPatientVisitsURLPath, patient_file.NewPatientVisitsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorPatientPharmacyURLPath, patient_file.NewDoctorUpdatePatientPharmacyHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorTreatmentPlansURLPath, doctor_treatment_plan.NewDoctorTreatmentPlanHandler(conf.DataAPI, conf.ERxAPI, conf.MediaStore, conf.Dispatcher, conf.ERxRoutingQueue, conf.ERxStatusQueue, conf.ERxRouting))
	authenticationRequired(conf, apipaths.DoctorTreatmentPlansListURLPath, doctor_treatment_plan.NewDeprecatedListHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorTPScheduledMessageURLPath, doctor_treatment_plan.NewScheduledMessageHandler(conf.DataAPI, conf.MediaStore, conf.Dispatcher))
	authenticationRequired(conf, apipaths.DoctorPharmacySearchURLPath, doctor.NewPharmacySearchHandler(conf.DataAPI, conf.ERxAPI))
	authenticationRequired(conf, apipaths.DoctorVisitReviewURLPath, patient_file.NewDoctorPatientVisitReviewHandler(conf.DataAPI, conf.Dispatcher, conf.MediaStore, conf.AuthTokenExpiration, conf.WebDomain))
	authenticationRequired(conf, apipaths.DoctorVisitDiagnosisListURLPath, diaghandlers.NewDiagnosisListHandler(conf.DataAPI, conf.DiagnosisAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.DoctorPatientCasesListURLPath, patient_file.NewPatientCaseListHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorDiagnosisURLPath, diaghandlers.NewDiagnosisHandler(conf.DataAPI, conf.DiagnosisAPI))
	authenticationRequired(conf, apipaths.DoctorDiagnosisSearchURLPath, diaghandlers.NewSearchHandler(conf.DataAPI, conf.DiagnosisAPI))
	authenticationRequired(conf, apipaths.DoctorSelectMedicationURLPath, doctor_treatment_plan.NewMedicationSelectHandler(conf.DataAPI, conf.ERxAPI))
	authenticationRequired(conf, apipaths.DoctorVisitTreatmentsURLPath, doctor_treatment_plan.NewTreatmentsHandler(conf.DataAPI, conf.ERxAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.DoctorMedicationSearchURLPath, handlers.NewAutocompleteHandler(conf.DataAPI, conf.ERxAPI))
	authenticationRequired(conf, apipaths.DoctorMedicationStrengthsURLPath, doctor_treatment_plan.NewMedicationStrengthSearchHandler(conf.DataAPI, conf.ERxAPI))
	authenticationRequired(conf, apipaths.DoctorMedicationDispenseUnitsURLPath, doctor_treatment_plan.NewMedicationDispenseUnitsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorRegimenURLPath, doctor_treatment_plan.NewRegimenHandler(conf.DataAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.DoctorSavedNoteURLPath, doctor_treatment_plan.NewSavedNoteHandler(conf.DataAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.DoctorCaseClaimURLPath, doctor_queue.NewClaimPatientCaseAccessHandler(conf.DataAPI, conf.AnalyticsLogger, conf.MetricsRegistry.Scope("doctor_queue")))
	authenticationRequired(conf, apipaths.DoctorAssignCaseURLPath, messages.NewAssignHandler(conf.DataAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.DoctorCaseCareTeamURLPath, patient_case.NewCareTeamHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorPatientFollowupURLPath, patient_file.NewFollowupHandler(conf.DataAPI, conf.AuthAPI, conf.AuthTokenExpiration, conf.Dispatcher))
	authenticationRequired(conf, apipaths.TPResourceGuideURLPath, doctor_treatment_plan.NewResourceGuideHandler(conf.DataAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.DoctorTokensURLPath, doctor_treatment_plan.NewDoctorTokensHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorPatientCapabilities, patient_file.NewPatientCapabilitiesHandler(conf.DataAPI, conf.AuthAPI, appFeatures))
	// Patient Feedback
	authenticationRequired(conf, apipaths.PatientFeedbackURLPath, patient.NewFeedbackHandler(conf.DataAPI, taggingClient, conf.Cfg))
	authenticationRequired(conf, apipaths.PatientFeedbackPromptURLPath, patient.NewFeedbackPromptHandler(conf.DataAPI))

	// Care Provider URLs
	noAuthenticationRequired(conf, apipaths.CareProviderSelectionURLPath, careprovider.NewSelectionHandler(conf.DataAPI, conf.APICDNDomain, conf.NumDoctorSelection))
	noAuthenticationRequired(conf, apipaths.CareProviderProfileURLPath, careprovider.NewProfileHandler(conf.DataAPI, conf.APICDNDomain))
	noAuthenticationRequired(conf, apipaths.CareProviderURLPath, careprovider.NewCareProviderHandler(conf.DataAPI, conf.APICDNDomain))

	// Tagging APIs
	authenticationRequired(conf, apipaths.TagURLPath, tagging.NewTagHandler(taggingClient))
	authenticationRequired(conf, apipaths.TagCaseMembershipURLPath, tagging.NewTagCaseMembershipHandler(taggingClient))
	authenticationRequired(conf, apipaths.TagCaseAssociationURLPath, tagging.NewTagCaseAssociationHandler(taggingClient))
	authenticationRequired(conf, apipaths.TagSavedSearchURLPath, tagging.NewTagSavedSearchHandler(taggingClient))

	// Miscellaneous APIs
	authenticationRequired(conf, apipaths.AppEventURLPath, appevent.NewHandler(conf.DataAPI, conf.Dispatcher))
	noAuthenticationRequired(conf, apipaths.PromotionsConfirmationURLPath, promotions.NewPromotionConfirmationHandler(conf.DataAPI, conf.AnalyticsLogger))
	authenticationRequired(conf, apipaths.PatienPromoCodeURLPath, promotions.NewPatientPromotionsHandler(conf.DataAPI, conf.AuthAPI, conf.AnalyticsLogger))
	authenticationRequired(conf, apipaths.ReferralsURLPath, promotions.NewReferralProgramHandler(conf.DataAPI, conf.WebDomain))

	mediaHandler := media.NewHandler(conf.DataAPI, conf.MediaStore, conf.Stores.MustGet("media-cache").(storage.DeterministicStore), conf.AuthTokenExpiration, conf.MetricsRegistry.Scope("media/handler"))
	noAuthenticationRequired(conf, apipaths.PhotoURLPath, mediaHandler)
	noAuthenticationRequired(conf, apipaths.MediaURLPath, mediaHandler)
	noAuthenticationRequired(conf, apipaths.NotifyMeURLPath,
		ratelimit.RemoteAddrHandler(
			handlers.NewNotifyMeHandler(conf.DataAPI),
			conf.RateLimiters.Get("login"),
			"notify-me",
			conf.MetricsRegistry))
	noAuthenticationRequired(conf, apipaths.PatientPathwaysURLPath, patient_visit.NewPathwayMenuHandler(conf.DataAPI))
	noAuthenticationRequired(conf, apipaths.PatientPathwayDetailsURLPath, patient_visit.NewPathwayDetailsHandler(conf.DataAPI, conf.APICDNDomain, conf.Cfg))
	noAuthenticationRequired(conf, apipaths.PingURLPath, handlers.NewPingHandler())
	noAuthenticationRequired(conf, apipaths.AnalyticsURLPath, apiservice.NewAnalyticsHandler(conf.Dispatcher, conf.MetricsRegistry.Scope("analytics.event.client")))
	noAuthenticationRequired(conf, apipaths.ResetPasswordURLPath, passreset.NewForgotPasswordHandler(conf.DataAPI, conf.AuthAPI, conf.EmailService, conf.WebDomain))
	noAuthenticationRequired(conf, apipaths.ProfileImageURLPath, handlers.NewProfileImageHandler(conf.DataAPI, conf.StaticResourceURL, conf.Stores.MustGet("thumbnails")))
	noAuthenticationRequired(conf, apipaths.SettingsURLPath, settings.NewHandler(conf.MinimumAppVersionConfigs))
	noAuthenticationRequired(conf, apipaths.PathwaySTPURLPath, patient_visit.NewPathwaySTPHandler(conf.DataAPI))

	// add the api to create demo visits to every environment except production
	if !environment.IsProd() {
		authenticationRequired(conf, apipaths.TrainingCasesURLPath, demo.NewTrainingCasesHandler(conf.DataAPI))
	}

	// Lazily include the feature set for the requesting app to the context
	handler := httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ctx = features.CtxWithSet(ctx, features.LazySet(func() features.Set {
			appInfo := apiservice.ExtractSpruceHeaders(r)
			return appFeatures.Set(strings.ToLower(appInfo.Platform.String()+"-"+appInfo.AppType), appInfo.AppVersion)
		}))
		conf.mux.ServeHTTP(ctx, w, r)
	})
	return conf.mux, handler
}

// Add an authenticated metriced handler to the mux
func authenticationRequired(conf *Config, path string, h httputil.ContextHandler) {
	conf.mux.Handle(path, apiservice.AuthenticationRequiredHandler(h, conf.AuthAPI))
}

// Add an unauthenticated metriced handler to the mux
func noAuthenticationRequired(conf *Config, path string, h httputil.ContextHandler) {
	conf.mux.Handle(path, apiservice.NoAuthenticationRequiredHandler(h, conf.AuthAPI))
}
