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
}

func New(conf *Config) http.Handler {
	// Initialize listneners
	doctor_queue.InitListeners(conf.DataAPI, conf.AnalyticsLogger, conf.Dispatcher, conf.NotificationManager, conf.MetricsRegistry.Scope("doctor_queue"), conf.JBCQMinutesThreshold, conf.CustomerSupportEmail)
	doctor_treatment_plan.InitListeners(conf.DataAPI, conf.Dispatcher)
	notify.InitListeners(conf.DataAPI, conf.Dispatcher)
	patient_case.InitListeners(conf.DataAPI, conf.Dispatcher, conf.NotificationManager)
	demo.InitListeners(conf.DataAPI, conf.Dispatcher, conf.APIDomain, conf.DosespotConfig.UserId)
	patient_visit.InitListeners(conf.DataAPI, conf.Dispatcher, conf.VisitQueue)
	doctor.InitListeners(conf.DataAPI, conf.Dispatcher)
	cost.InitListeners(conf.DataAPI, conf.Dispatcher)
	auth.InitListeners(conf.AuthAPI, conf.Dispatcher)

	mux := apiservice.NewAuthServeMux(conf.AuthAPI, conf.AnalyticsLogger, conf.MetricsRegistry.Scope("restapi"))

	addressValidationAPI := address.NewAddressValidationWithCacheWrapper(conf.AddressValidationAPI, conf.MaxCachedItems)

	// Patient/Doctor: Push notification APIs
	mux.Handle(apipaths.NotificationTokenURLPath, notify.NewNotificationHandler(conf.DataAPI, conf.NotifyConfigs, conf.SNSClient))
	mux.Handle(apipaths.NotificationPromptStatusURLPath, notify.NewPromptStatusHandler(conf.DataAPI))

	// Patient: Account related APIs
	mux.Handle(apipaths.PatientSignupURLPath, patient.NewSignupHandler(conf.DataAPI, conf.AuthAPI, conf.AnalyticsLogger, conf.Dispatcher, conf.AuthTokenExpiration,
		conf.Stores.MustGet("media"), conf.RateLimiters.Get("patient-signup"), addressValidationAPI,
		conf.MetricsRegistry.Scope("patient.signup")))
	mux.Handle(apipaths.PatientInfoURLPath, patient.NewUpdateHandler(conf.DataAPI))
	mux.Handle(apipaths.PatientAddressURLPath, patient.NewAddressHandler(conf.DataAPI, patient.BILLING_ADDRESS_TYPE))
	mux.Handle(apipaths.PatientPharmacyURLPath, patient.NewPharmacyHandler(conf.DataAPI))
	mux.Handle(apipaths.PatientAlertsURLPath, patient_file.NewAlertsHandler(conf.DataAPI))
	mux.Handle(apipaths.PatientIsAuthenticatedURLPath, handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	mux.Handle(apipaths.PatientCardURLPath, patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidationAPI))
	mux.Handle(apipaths.PatientDefaultCardURLPath, patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidationAPI))
	mux.Handle(apipaths.PatientAuthenticateURLPath, patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.Dispatcher,
		conf.StaticContentURL, conf.RateLimiters.Get("login"), conf.MetricsRegistry.Scope("patient.auth")))
	mux.Handle(apipaths.PatientRequestMedicalRecordURLPath, medrecord.NewRequestAPIHandler(conf.DataAPI, conf.MedicalRecordQueue))
	mux.Handle(apipaths.LogoutURLPath, patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.Dispatcher, conf.StaticContentURL,
		conf.RateLimiters.Get("login"), conf.MetricsRegistry.Scope("patient.auth")))
	mux.Handle(apipaths.PatientPCPURLPath, patient.NewPCPHandler(conf.DataAPI))
	mux.Handle(apipaths.PatientEmergencyContactsURLPath, patient.NewEmergencyContactsHandler(conf.DataAPI))
	mux.Handle(apipaths.PatientMeURLPath, patient.NewMeHandler(conf.DataAPI, conf.Dispatcher))
	mux.Handle(apipaths.PatientCareTeamURLPath, patient.NewCareTeamHandler(conf.DataAPI))
	mux.Handle(apipaths.PatientCostURLPath, cost.NewCostHandler(conf.DataAPI, conf.AnalyticsLogger))
	mux.Handle(apipaths.PatientCreditsURLPath, promotions.NewPatientCreditsHandler(conf.DataAPI))

	// Patient: Patient Case Related APIs
	mux.Handle(apipaths.CheckEligibilityURLPath, patient.NewCheckCareProvidingEligibilityHandler(conf.DataAPI, addressValidationAPI, conf.AnalyticsLogger))
	mux.Handle(apipaths.PatientVisitURLPath, patient.NewPatientVisitHandler(conf.DataAPI, conf.AuthAPI, conf.PaymentAPI, conf.AddressValidationAPI, conf.Dispatcher, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))
	mux.Handle(apipaths.PatientVisitsListURLPath, patient.NewVisitsListHandler(conf.DataAPI, conf.Dispatcher, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))
	mux.Handle(apipaths.PatientVisitIntakeURLPath, patient_visit.NewAnswerIntakeHandler(conf.DataAPI))
	mux.Handle(apipaths.PatientVisitMessageURLPath, patient_visit.NewMessageHandler(conf.DataAPI))
	mux.Handle(apipaths.PatientVisitPhotoAnswerURLPath, patient_visit.NewPhotoAnswerIntakeHandler(conf.DataAPI))
	mux.Handle(apipaths.PatientTreatmentsURLPath, treatment_plan.NewTreatmentsHandler(conf.DataAPI))

	mux.Handle(apipaths.TreatmentPlanURLPath, treatment_plan.NewTreatmentPlanHandler(conf.DataAPI))
	mux.Handle(apipaths.TreatmentGuideURLPath, treatment_plan.NewTreatmentGuideHandler(conf.DataAPI))
	mux.Handle(apipaths.AutocompleteURLPath, handlers.NewAutocompleteHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(apipaths.PharmacySearchURLPath, patient.NewPharmacySearchHandler(conf.DataAPI, conf.PharmacySearchAPI))

	// Patient: Home APIs
	mux.Handle(apipaths.PatientHomeURLPath, patient_case.NewHomeHandler(conf.DataAPI, conf.AuthAPI, conf.APIDomain, addressValidationAPI))
	mux.Handle(apipaths.PatientHowFAQURLPath, handlers.NewPatientFAQHandler(conf.StaticContentURL))
	mux.Handle(apipaths.PatientPricingFAQURLPath, handlers.NewPricingFAQHandler(conf.StaticContentURL))
	mux.Handle(apipaths.PatientFeaturedDoctorsURLPath, handlers.NewFeaturedDoctorsHandler(conf.StaticContentURL))

	//Patient/Doctor: Case APIs
	mux.Handle(apipaths.PatientCasesListURLPath, patient_case.NewListHandler(conf.DataAPI))
	mux.Handle(apipaths.PatientCasesURLPath, patient_case.NewCaseInfoHandler(conf.DataAPI))
	// Patient: Case APIs
	mux.Handle(apipaths.PatientCaseNotificationsURLPath, patient_case.NewNotificationsListHandler(conf.DataAPI, conf.APIDomain))

	// Patient/Doctor: Resource guide APIs
	mux.Handle(apipaths.ResourceGuideURLPath, reslib.NewHandler(conf.DataAPI))
	mux.Handle(apipaths.ResourceGuidesListURLPath, reslib.NewListHandler(conf.DataAPI))

	// Patient/Doctor: Message APIs
	mux.Handle(apipaths.CaseMessagesURLPath, messages.NewHandler(conf.DataAPI, conf.Dispatcher))
	mux.Handle(apipaths.CaseMessagesListURLPath, messages.NewListHandler(conf.DataAPI, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))

	// Doctor: Account APIs
	mux.Handle(apipaths.DoctorSignupURLPath, doctor.NewSignupDoctorHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle(apipaths.DoctorAuthenticateURLPath, doctor.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.SMSAPI, conf.Dispatcher,
		conf.SMSFromNumber, conf.TwoFactorExpiration, conf.RateLimiters.Get("login"), conf.MetricsRegistry.Scope("doctor.auth")))
	mux.Handle(apipaths.DoctorAuthenticateTwoFactorURLPath, doctor.NewTwoFactorHandler(conf.DataAPI, conf.AuthAPI, conf.SMSAPI, conf.SMSFromNumber, conf.TwoFactorExpiration))
	mux.Handle(apipaths.DoctorIsAuthenticatedURLPath, handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	mux.Handle(apipaths.DoctorQueueURLPath, doctor_queue.NewQueueHandler(conf.DataAPI))

	// Doctor: Prescription related APIs
	mux.Handle(apipaths.DoctorRXErrorURLPath, doctor.NewPrescriptionErrorHandler(conf.DataAPI))
	mux.Handle(apipaths.DoctorRXErrorResolveURLPath, doctor.NewPrescriptionErrorIgnoreHandler(conf.DataAPI, conf.ERxAPI, conf.Dispatcher))
	mux.Handle(apipaths.DoctorRefillRxURLPath, doctor.NewRefillRxHandler(conf.DataAPI, conf.ERxAPI, conf.Dispatcher, conf.ERxStatusQueue))
	mux.Handle(apipaths.DoctorRefillRxDenialReasonsURLPath, doctor.NewRefillRxDenialReasonsHandler(conf.DataAPI))
	mux.Handle(apipaths.DoctorFTPURLPath, doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(conf.DataAPI))
	mux.Handle(apipaths.DoctorManageFTPURLPath, doctor_treatment_plan.NewManageFTPHandler(conf.DataAPI))
	mux.Handle(apipaths.DoctorTreatmentTemplatesURLPath, doctor_treatment_plan.NewTreatmentTemplatesHandler(conf.DataAPI))

	// Doctor: Patient file APIs
	mux.Handle(apipaths.DoctorPatientTreatmentsURLPath, patient_file.NewDoctorPatientTreatmentsHandler(conf.DataAPI))
	mux.Handle(apipaths.DoctorPatientInfoURLPath, patient_file.NewDoctorPatientHandler(conf.DataAPI, conf.ERxAPI, conf.AddressValidationAPI))
	mux.Handle(apipaths.DoctorPatientAppInfoURLPath, patient_file.NewPatientAppInfoHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle(apipaths.DoctorPatientVisitsURLPath, patient_file.NewPatientVisitsHandler(conf.DataAPI))
	mux.Handle(apipaths.DoctorPatientPharmacyURLPath, patient_file.NewDoctorUpdatePatientPharmacyHandler(conf.DataAPI))
	mux.Handle(apipaths.DoctorTreatmentPlansURLPath, doctor_treatment_plan.NewDoctorTreatmentPlanHandler(conf.DataAPI, conf.ERxAPI, conf.Dispatcher, conf.ERxRoutingQueue, conf.ERxStatusQueue, conf.ERxRouting))
	mux.Handle(apipaths.DoctorTreatmentPlansListURLPath, doctor_treatment_plan.NewListHandler(conf.DataAPI))
	mux.Handle(apipaths.DoctorPharmacySearchURLPath, doctor.NewPharmacySearchHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(apipaths.DoctorVisitReviewURLPath, patient_file.NewDoctorPatientVisitReviewHandler(conf.DataAPI, conf.Dispatcher, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))
	mux.Handle(apipaths.DoctorVisitDiagnosisURLPath, patient_visit.NewDiagnosePatientHandler(conf.DataAPI, conf.AuthAPI, conf.Dispatcher))
	mux.Handle(apipaths.DoctorSelectMedicationURLPath, doctor_treatment_plan.NewMedicationSelectHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(apipaths.DoctorVisitTreatmentsURLPath, doctor_treatment_plan.NewTreatmentsHandler(conf.DataAPI, conf.ERxAPI, conf.Dispatcher))
	mux.Handle(apipaths.DoctorMedicationSearchURLPath, handlers.NewAutocompleteHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(apipaths.DoctorMedicationStrengthsURLPath, doctor_treatment_plan.NewMedicationStrengthSearchHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(apipaths.DoctorMedicationDispenseUnitsURLPath, doctor_treatment_plan.NewMedicationDispenseUnitsHandler(conf.DataAPI))
	mux.Handle(apipaths.DoctorRegimenURLPath, doctor_treatment_plan.NewRegimenHandler(conf.DataAPI, conf.Dispatcher))
	mux.Handle(apipaths.DoctorSavedNoteURLPath, doctor_treatment_plan.NewSavedNoteHandler(conf.DataAPI, conf.Dispatcher))
	mux.Handle(apipaths.DoctorCaseClaimURLPath, doctor_queue.NewClaimPatientCaseAccessHandler(conf.DataAPI, conf.AnalyticsLogger, conf.MetricsRegistry.Scope("doctor_queue")))
	mux.Handle(apipaths.DoctorAssignCaseURLPath, messages.NewAssignHandler(conf.DataAPI, conf.Dispatcher))
	mux.Handle(apipaths.DoctorCaseCareTeamURLPath, patient_case.NewCareTeamHandler(conf.DataAPI))
	mux.Handle(apipaths.DoctorPatientFollowupURLPath, patient_file.NewFollowupHandler(conf.DataAPI, conf.AuthAPI, conf.AuthTokenExpiration, conf.Dispatcher, conf.Stores.MustGet("media")))

	// Miscellaneous APIs
	mux.Handle(apipaths.ContentURLPath, handlers.NewStaticContentHandler(conf.DataAPI, conf.CloudStorageAPI, conf.ContentBucket, conf.AWSRegion))
	mux.Handle(apipaths.PingURLPath, handlers.NewPingHandler())
	mux.Handle(apipaths.PhotoURLPath, media.NewHandler(conf.DataAPI, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))
	mux.Handle(apipaths.MediaURLPath, media.NewHandler(conf.DataAPI, conf.Stores.MustGet("media"), conf.AuthTokenExpiration))
	mux.Handle(apipaths.LayoutUploadURLPath, layout.NewLayoutUploadHandler(conf.DataAPI))
	mux.Handle(apipaths.AppEventURLPath, app_event.NewHandler(conf.Dispatcher))
	mux.Handle(apipaths.AnalyticsURLPath, apiservice.NewAnalyticsHandler(conf.AnalyticsLogger, conf.MetricsRegistry.Scope("analytics.event.client")))
	mux.Handle(apipaths.ResetPasswordURLPath, passreset.NewForgotPasswordHandler(conf.DataAPI, conf.AuthAPI, conf.EmailService, conf.CustomerSupportEmail, conf.WebDomain))
	mux.Handle(apipaths.CareProviderProfileURLPath, handlers.NewCareProviderProfileHandler(conf.DataAPI))
	mux.Handle(apipaths.ThumbnailURLPath, handlers.NewThumbnailHandler(conf.DataAPI, conf.StaticResourceURL, conf.Stores.MustGet("thumbnails")))
	mux.Handle(apipaths.SettingsURLPath, settings.NewHandler(conf.MinimumAppVersionConfigs))
	mux.Handle(apipaths.PromotionsURLPath, promotions.NewPromotionsHandler(conf.DataAPI))
	mux.Handle(apipaths.ReferralProgramsTemplateURLPath, promotions.NewReferralProgramTemplateHandler(conf.DataAPI))
	mux.Handle(apipaths.ReferralsURLPath, promotions.NewReferralProgramHandler(conf.DataAPI, conf.WebDomain))
	// add the api to create demo visits to every environment except production
	if !environment.IsProd() {
		mux.Handle(apipaths.TrainingCasesURLPath, demo.NewTrainingCasesHandler(conf.DataAPI))
	}

	return mux
}
