package router

import (
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/auth"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/demo"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/layout"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/ratelimit"
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
	"github.com/sprucehealth/backend/treatment_plan"
)

type Config struct {
	DataAPI                  api.DataAPI
	AuthAPI                  api.AuthAPI
	Dispatcher               *dispatch.Dispatcher
	AuthTokenExpiration      time.Duration
	AddressValidationAPI     address.AddressValidationAPI
	PharmacySearchAPI        pharmacy.PharmacySearchAPI
	SNSClient                sns.SNSService
	PaymentAPI               apiservice.StripeClient
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
	CloudStorageAPI          api.CloudStorageAPI
	Stores                   storage.StoreMap
	RateLimiters             ratelimit.KeyedRateLimiters
	AnalyticsLogger          analytics.Logger
	ERxRouting               bool
	JBCQMinutesThreshold     int
	MaxCachedItems           int
	CustomerSupportEmail     string
	TechnicalSupportEmail    string
	APIDomain                string
	WebDomain                string
	StaticContentURL         string
	StaticResourceURL        string
	ContentBucket            string
	AWSRegion                string
	TwoFactorExpiration      int
	SMSFromNumber            string

	mux *muxWithRegisteredPaths
}

func New(conf *Config) http.Handler {
	// Initialize listneners
	doctor_queue.InitListeners(conf.DataAPI, conf.AnalyticsLogger, conf.Dispatcher, conf.NotificationManager, conf.MetricsRegistry.Scope("doctor_queue"), conf.JBCQMinutesThreshold, conf.CustomerSupportEmail)
	doctor_treatment_plan.InitListeners(conf.DataAPI, conf.Dispatcher)
	notify.InitListeners(conf.DataAPI, conf.Dispatcher)
	patient_case.InitListeners(conf.DataAPI, conf.Dispatcher, conf.NotificationManager)
	demo.InitListeners(conf.DataAPI, conf.Dispatcher, conf.APIDomain, conf.DosespotConfig.UserID)
	patient_visit.InitListeners(conf.DataAPI, conf.Dispatcher, conf.VisitQueue)
	doctor.InitListeners(conf.DataAPI, conf.Dispatcher)
	cost.InitListeners(conf.DataAPI, conf.Dispatcher)
	auth.InitListeners(conf.AuthAPI, conf.Dispatcher)

	conf.mux = newMux()

	addressValidationAPI := address.NewAddressValidationWithCacheWrapper(conf.AddressValidationAPI, conf.MaxCachedItems)

	// Patient/Doctor: Push notification APIs
	authenticationRequired(conf, apipaths.NotificationTokenURLPath, notify.NewNotificationHandler(conf.DataAPI, conf.NotifyConfigs, conf.SNSClient))
	authenticationRequired(conf, apipaths.NotificationPromptStatusURLPath, notify.NewPromptStatusHandler(conf.DataAPI))

	// Patient: Account related APIs

	authenticationRequired(conf, apipaths.PatientInfoURLPath, patient.NewUpdateHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientAddressURLPath, patient.NewAddressHandler(conf.DataAPI, patient.BILLING_ADDRESS_TYPE))
	authenticationRequired(conf, apipaths.PatientPharmacyURLPath, patient.NewPharmacyHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientAlertsURLPath, patient_file.NewAlertsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientIsAuthenticatedURLPath, handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	authenticationRequired(conf, apipaths.PatientCardURLPath, patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidationAPI))
	authenticationRequired(conf, apipaths.PatientDefaultCardURLPath, patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidationAPI))
	authenticationRequired(conf, apipaths.PatientRequestMedicalRecordURLPath, medrecord.NewRequestAPIHandler(conf.DataAPI, conf.MedicalRecordQueue))
	authenticationRequired(conf, apipaths.LogoutURLPath, patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.Dispatcher, conf.StaticContentURL,
		conf.RateLimiters.Get("login"), conf.MetricsRegistry.Scope("patient.auth")))
	authenticationRequired(conf, apipaths.PatientPCPURLPath, patient.NewPCPHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientEmergencyContactsURLPath, patient.NewEmergencyContactsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientMeURLPath, patient.NewMeHandler(conf.DataAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.PatientCareTeamURLPath, patient.NewCareTeamHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientCostURLPath, cost.NewCostHandler(conf.DataAPI, conf.AnalyticsLogger))
	authenticationRequired(conf, apipaths.PatientCreditsURLPath, promotions.NewPatientCreditsHandler(conf.DataAPI))
	noAuthenticationRequired(conf, apipaths.PatientSignupURLPath, patient.NewSignupHandler(
		conf.DataAPI, conf.AuthAPI, conf.AnalyticsLogger, conf.Dispatcher, conf.AuthTokenExpiration,
		conf.Stores.MustGet("media"), conf.RateLimiters.Get("patient-signup"), addressValidationAPI,
		conf.MetricsRegistry.Scope("patient.signup")))
	noAuthenticationRequired(conf, apipaths.PatientAuthenticateURLPath, patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.Dispatcher,
		conf.StaticContentURL, conf.RateLimiters.Get("login"), conf.MetricsRegistry.Scope("patient.auth")))

	// Patient: Patient Case Related APIs
	authenticationRequired(conf, apipaths.PatientVisitURLPath, patient.NewPatientVisitHandler(conf.DataAPI, conf.AuthAPI, conf.PaymentAPI, conf.AddressValidationAPI, conf.Dispatcher, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))
	authenticationRequired(conf, apipaths.PatientVisitsListURLPath, patient.NewVisitsListHandler(conf.DataAPI, conf.Dispatcher, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))
	authenticationRequired(conf, apipaths.PatientVisitIntakeURLPath, patient_visit.NewAnswerIntakeHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientVisitMessageURLPath, patient_visit.NewMessageHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientVisitPhotoAnswerURLPath, patient_visit.NewPhotoAnswerIntakeHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientTreatmentsURLPath, treatment_plan.NewTreatmentsHandler(conf.DataAPI))
	noAuthenticationRequired(conf, apipaths.CheckEligibilityURLPath, patient.NewCheckCareProvidingEligibilityHandler(conf.DataAPI, addressValidationAPI, conf.AnalyticsLogger))

	authenticationRequired(conf, apipaths.TreatmentPlanURLPath, treatment_plan.NewTreatmentPlanHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.TreatmentGuideURLPath, treatment_plan.NewTreatmentGuideHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.AutocompleteURLPath, handlers.NewAutocompleteHandler(conf.DataAPI, conf.ERxAPI))
	authenticationRequired(conf, apipaths.PharmacySearchURLPath, patient.NewPharmacySearchHandler(conf.DataAPI, conf.PharmacySearchAPI))

	// Patient: Home APIs
	noAuthenticationRequired(conf, apipaths.PatientHomeURLPath, patient_case.NewHomeHandler(conf.DataAPI, conf.AuthAPI, conf.APIDomain, addressValidationAPI))
	noAuthenticationRequired(conf, apipaths.PatientHowFAQURLPath, handlers.NewPatientFAQHandler(conf.StaticContentURL))
	noAuthenticationRequired(conf, apipaths.PatientPricingFAQURLPath, handlers.NewPricingFAQHandler(conf.StaticContentURL))
	noAuthenticationRequired(conf, apipaths.PatientFeaturedDoctorsURLPath, handlers.NewFeaturedDoctorsHandler(conf.StaticContentURL))

	// Patient/Doctor: Case APIs
	authenticationRequired(conf, apipaths.PatientCasesListURLPath, patient_case.NewListHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.PatientCasesURLPath, patient_case.NewCaseInfoHandler(conf.DataAPI))
	// Patient: Case APIs
	authenticationRequired(conf, apipaths.PatientCaseNotificationsURLPath, patient_case.NewNotificationsListHandler(conf.DataAPI, conf.APIDomain))

	// Patient/Doctor: Resource guide APIs
	noAuthenticationRequired(conf, apipaths.ResourceGuideURLPath, reslib.NewHandler(conf.DataAPI))
	noAuthenticationRequired(conf, apipaths.ResourceGuidesListURLPath, reslib.NewListHandler(conf.DataAPI))

	// Patient/Doctor: Message APIs
	authenticationRequired(conf, apipaths.CaseMessagesURLPath, messages.NewHandler(conf.DataAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.CaseMessagesListURLPath, messages.NewListHandler(conf.DataAPI, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))

	// Doctor: Account APIs
	authenticationRequired(conf, apipaths.DoctorIsAuthenticatedURLPath, handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	authenticationRequired(conf, apipaths.DoctorQueueURLPath, doctor_queue.NewQueueHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorCaseHistoryURLPath, doctor_queue.NewPatientsFeedHandler(conf.DataAPI))
	noAuthenticationRequired(conf, apipaths.DoctorSignupURLPath, doctor.NewSignupDoctorHandler(conf.DataAPI, conf.AuthAPI))
	noAuthenticationRequired(conf, apipaths.DoctorAuthenticateURLPath, doctor.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.SMSAPI, conf.Dispatcher,
		conf.SMSFromNumber, conf.TwoFactorExpiration, conf.RateLimiters.Get("login"), conf.MetricsRegistry.Scope("doctor.auth")))
	noAuthenticationRequired(conf, apipaths.DoctorAuthenticateTwoFactorURLPath, doctor.NewTwoFactorHandler(conf.DataAPI, conf.AuthAPI, conf.SMSAPI, conf.SMSFromNumber, conf.TwoFactorExpiration))

	// Doctor: Prescription related APIs
	authenticationRequired(conf, apipaths.DoctorRXErrorURLPath, doctor.NewPrescriptionErrorHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorRXErrorResolveURLPath, doctor.NewPrescriptionErrorIgnoreHandler(conf.DataAPI, conf.ERxAPI, conf.Dispatcher))
	authenticationRequired(conf, apipaths.DoctorRefillRxURLPath, doctor.NewRefillRxHandler(conf.DataAPI, conf.ERxAPI, conf.Dispatcher, conf.ERxStatusQueue))
	authenticationRequired(conf, apipaths.DoctorRefillRxDenialReasonsURLPath, doctor.NewRefillRxDenialReasonsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorFTPURLPath, doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorManageFTPURLPath, doctor_treatment_plan.NewManageFTPHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorTreatmentTemplatesURLPath, doctor_treatment_plan.NewTreatmentTemplatesHandler(conf.DataAPI))

	// Doctor: Patient file APIs
	authenticationRequired(conf, apipaths.DoctorPatientTreatmentsURLPath, patient_file.NewDoctorPatientTreatmentsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorPatientInfoURLPath, patient_file.NewDoctorPatientHandler(conf.DataAPI, conf.ERxAPI, conf.AddressValidationAPI))
	authenticationRequired(conf, apipaths.DoctorPatientAppInfoURLPath, patient_file.NewPatientAppInfoHandler(conf.DataAPI, conf.AuthAPI))
	authenticationRequired(conf, apipaths.DoctorPatientVisitsURLPath, patient_file.NewPatientVisitsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorPatientCasesListURLPath, patient_file.NewPatientCaseListHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorPatientPharmacyURLPath, patient_file.NewDoctorUpdatePatientPharmacyHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorTreatmentPlansURLPath, doctor_treatment_plan.NewDoctorTreatmentPlanHandler(conf.DataAPI, conf.ERxAPI, conf.Dispatcher, conf.ERxRoutingQueue, conf.ERxStatusQueue, conf.ERxRouting))
	authenticationRequired(conf, apipaths.DoctorTreatmentPlansListURLPath, doctor_treatment_plan.NewListHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.DoctorPharmacySearchURLPath, doctor.NewPharmacySearchHandler(conf.DataAPI, conf.ERxAPI))
	authenticationRequired(conf, apipaths.DoctorVisitReviewURLPath, patient_file.NewDoctorPatientVisitReviewHandler(conf.DataAPI, conf.Dispatcher, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))
	authenticationRequired(conf, apipaths.DoctorVisitDiagnosisURLPath, patient_visit.NewDiagnosePatientHandler(conf.DataAPI, conf.AuthAPI, conf.Dispatcher))
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
	authenticationRequired(conf, apipaths.DoctorPatientFollowupURLPath, patient_file.NewFollowupHandler(conf.DataAPI, conf.AuthAPI, conf.AuthTokenExpiration, conf.Dispatcher, conf.Stores.MustGet("media")))

	// Miscellaneous APIs
	authenticationRequired(conf, apipaths.PhotoURLPath, media.NewHandler(conf.DataAPI, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))
	authenticationRequired(conf, apipaths.MediaURLPath, media.NewHandler(conf.DataAPI, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))
	authenticationRequired(conf, apipaths.LayoutUploadURLPath, layout.NewLayoutUploadHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.AppEventURLPath, app_event.NewHandler(conf.Dispatcher))
	authenticationRequired(conf, apipaths.PromotionsURLPath, promotions.NewPromotionsHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.ReferralProgramsTemplateURLPath, promotions.NewReferralProgramTemplateHandler(conf.DataAPI))
	authenticationRequired(conf, apipaths.ReferralsURLPath, promotions.NewReferralProgramHandler(conf.DataAPI, conf.WebDomain))
	noAuthenticationRequired(conf, apipaths.ContentURLPath, handlers.NewStaticContentHandler(conf.DataAPI, conf.CloudStorageAPI, conf.ContentBucket, conf.AWSRegion))
	noAuthenticationRequired(conf, apipaths.PingURLPath, handlers.NewPingHandler())
	noAuthenticationRequired(conf, apipaths.AnalyticsURLPath, apiservice.NewAnalyticsHandler(conf.AnalyticsLogger, conf.MetricsRegistry.Scope("analytics.event.client")))
	noAuthenticationRequired(conf, apipaths.ResetPasswordURLPath, passreset.NewForgotPasswordHandler(conf.DataAPI, conf.AuthAPI, conf.EmailService, conf.CustomerSupportEmail, conf.WebDomain))
	noAuthenticationRequired(conf, apipaths.CareProviderProfileURLPath, handlers.NewCareProviderProfileHandler(conf.DataAPI))
	noAuthenticationRequired(conf, apipaths.ThumbnailURLPath, handlers.NewThumbnailHandler(conf.DataAPI, conf.StaticResourceURL, conf.Stores.MustGet("thumbnails")))
	noAuthenticationRequired(conf, apipaths.SettingsURLPath, settings.NewHandler(conf.MinimumAppVersionConfigs))

	// add the api to create demo visits to every environment except production
	if !environment.IsProd() {
		authenticationRequired(conf, apipaths.TrainingCasesURLPath, demo.NewTrainingCasesHandler(conf.DataAPI))
	}

	// DEPRECATED: Remove after Buzz Lightyear release
	authenticationRequired(conf, apipaths.DeprecatedDoctorSavedMessagesURLPath, doctor_treatment_plan.NewSavedNoteCompatibilityHandler(conf.DataAPI))

	return apiservice.MetricsHandler(
		conf.mux,
		conf.AnalyticsLogger,
		conf.MetricsRegistry.Scope("restapi"))
}

func authenticationRequired(conf *Config, path string, h http.Handler) {
	conf.mux.Handle(
		path,
		apiservice.AuthenticationRequiredHandler(
			h,
			conf.AuthAPI,
		),
	)
}

func noAuthenticationRequired(conf *Config, path string, h http.Handler) {
	conf.mux.Handle(
		path,
		apiservice.NoAuthenticationRequiredHandler(
			h,
		),
	)
}
