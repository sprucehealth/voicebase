package router

import (
	"net/http"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/demo"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/layout"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/payment"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/misc/handlers"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/passreset"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_case"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/photos"
	"github.com/sprucehealth/backend/reslib"
	"github.com/sprucehealth/backend/support"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/third_party/github.com/subosito/twilio"
	"github.com/sprucehealth/backend/treatment_plan"
)

const (
	AnalyticsURLPath                     = "/v1/event/client"
	AppEventURLPath                      = "/v1/app_event"
	AutocompleteURLPath                  = "/v1/autocomplete"
	CareProviderProfileURLPath           = "/v1/care_provider_profile"
	CaseMessagesListURLPath              = "/v1/case/messages/list"
	CaseMessagesURLPath                  = "/v1/case/messages"
	CheckEligibilityURLPath              = "/v1/check_eligibility"
	ContentURLPath                       = "/v1/content"
	DoctorAdviceURLPath                  = "/v1/doctor/visit/advice"
	DoctorAssignCaseURLPath              = "/v1/doctor/case/assign"
	DoctorAuthenticateURLPath            = "/v1/doctor/authenticate"
	DoctorCaseCareTeamURLPath            = "/v1/doctor/case/care_team"
	DoctorCaseClaimURLPath               = "/v1/doctor/patient/case/claim"
	DoctorFTPURLPath                     = "/v1/doctor/favorite_treatment_plans"
	DoctorIsAuthenticatedURLPath         = "/v1/doctor/isauthenticated"
	DoctorMedicationDispenseUnitsURLPath = "/v1/doctor/visit/treatment/medication_dispense_units"
	DoctorMedicationSearchURLPath        = "/v1/doctor/visit/treatment/medication_suggestions"
	DoctorMedicationStrengthsURLPath     = "/v1/doctor/visit/treatment/medication_strengths"
	DoctorPatientInfoURLPath             = "/v1/doctor/patient"
	DoctorPatientPharmacyURLPath         = "/v1/doctor/patient/pharmacy"
	DoctorPatientTreatmentsURLPath       = "/v1/doctor/patient/treatments"
	DoctorPatientVisitsURLPath           = "/v1/doctor/patient/visits"
	DoctorPharmacySearchURLPath          = "/v1/doctor/pharmacy"
	DoctorQueueURLPath                   = "/v1/doctor/queue"
	DoctorRefillRxDenialReasonsURLPath   = "/v1/doctor/rx/refill/denial_reasons"
	DoctorRefillRxURLPath                = "/v1/doctor/rx/refill/request"
	DoctorRegimenURLPath                 = "/v1/doctor/visit/regimen"
	DoctorRXErrorResolveURLPath          = "/v1/doctor/rx/error/resolve"
	DoctorRXErrorURLPath                 = "/v1/doctor/rx/error"
	DoctorSavedMessagesURLPath           = "/v1/doctor/saved_messages"
	DoctorSelectMedicationURLPath        = "/v1/doctor/visit/treatment/new"
	DoctorSignupURLPath                  = "/v1/doctor/signup"
	DoctorTreatmentPlansListURLPath      = "/v1/doctor/treatment_plans/list"
	DoctorTreatmentPlansURLPath          = "/v1/doctor/treatment_plans"
	DoctorTreatmentTemplatesURLPath      = "/v1/doctor/treatment/templates"
	DoctorVisitDiagnosisURLPath          = "/v1/doctor/visit/diagnosis"
	DoctorVisitReviewURLPath             = "/v1/doctor/visit/review"
	DoctorVisitTreatmentsURLPath         = "/v1/doctor/visit/treatment/treatments"
	LayoutUploadURLPath                  = "/v1/layous/upload"
	LogoutURLPath                        = "/v1/logout"
	NotificationPromptStatusURLPath      = "/v1/notification/prompt_status"
	NotificationTokenURLPath             = "/v1/notification/token"
	PatientAddressURLPath                = "/v1/patient/address/billing"
	PatientAlertsURLPath                 = "/v1/patient/alerts"
	PatientAuthenticateURLPath           = "/v1/authenticate"
	PatientCardURLPath                   = "/v1/credit_card"
	PatientCaseNotificationsURLPath      = "/v1/patient/case/notifications"
	PatientCasesListURLPath              = "/v1/cases/list"
	PatientCasesURLPath                  = "/v1/cases"
	PatientDefaultCardURLPath            = "/v1/credit_card/default"
	PatientEmergencyContactsURLPath      = "/v1/patient/emergency_contacts"
	PatientHomeURLPath                   = "/v1/patient/home"
	PatientInfoURLPath                   = "/v1/patient/info"
	PatientIsAuthenticatedURLPath        = "/v1/patient/isauthenticated"
	PatientPCPURLPath                    = "/v1/patient/pcp"
	PatientPharmacyURLPath               = "/v1/patient/pharmacy"
	PatientSignupURLPath                 = "/v1/patient"
	PatientTreatmentsURLPath             = "/v1/patient/treatments"
	PatientVisitIntakeURLPath            = "/v1/patient/visit/answer"
	PatientVisitPhotoAnswerURLPath       = "/v1/patient/visit/photo_answer"
	PatientVisitURLPath                  = "/v1/patient/visit"
	PharmacySearchURLPath                = "/v1/pharmacy_search"
	PhotoURLPath                         = "/v1/photo"
	PingURLPath                          = "/v1/ping"
	ResetPasswordURLPath                 = "/v1/reset_password"
	ResourceGuidesListURLPath            = "/v1/resourceguide/list"
	ResourceGuideURLPath                 = "/v1/resourceguide"
	TreatmentGuideURLPath                = "/v1/treatment_guide"
	TreatmentPlanURLPath                 = "/v1/treatment_plan"
)

type Config struct {
	DataAPI                  api.DataAPI
	AuthAPI                  api.AuthAPI
	AddressValidationAPI     address.AddressValidationAPI
	ZipcodeToCityStateMapper map[string]*address.CityState
	PharmacySearchAPI        pharmacy.PharmacySearchAPI
	SNSClient                sns.SNSService
	PaymentAPI               payment.PaymentAPI
	NotifyConfigs            *config.NotificationConfigs
	NotificationManager      *notify.NotificationManager
	ERxStatusQueue           *common.SQSQueue
	ERxAPI                   erx.ERxAPI
	EmailService             email.Service
	MetricsRegistry          metrics.Registry
	TwilioClient             *twilio.Client
	CloudStorageAPI          api.CloudStorageAPI
	Stores                   map[string]storage.Store
	AnalyticsLogger          analytics.Logger
	ERxRouting               bool
	JBCQMinutesThreshold     int
	MaxCachedItems           int
	CustomerSupportEmail     string
	TechnicalSupportEmail    string
	APISubdomain             string
	WebSubdomain             string
	StaticContentURL         string
	ContentBucket            string
	AWSRegion                string
}

func New(conf *Config) http.Handler {

	// Initialize listneners
	doctor_queue.InitListeners(conf.DataAPI, conf.NotificationManager, conf.MetricsRegistry.Scope("doctor_queue"), conf.JBCQMinutesThreshold)
	doctor_treatment_plan.InitListeners(conf.DataAPI)
	notify.InitListeners(conf.DataAPI)
	support.InitListeners(conf.TechnicalSupportEmail, conf.CustomerSupportEmail, conf.NotificationManager)
	patient_case.InitListeners(conf.DataAPI, conf.NotificationManager)
	patient_visit.InitListeners(conf.DataAPI)
	demo.InitListeners(conf.DataAPI, conf.APISubdomain)

	mux := apiservice.NewAuthServeMux(conf.AuthAPI, conf.MetricsRegistry.Scope("restapi"))

	addressValidationWithCacheAndHack := address.NewHackAddressValidationWrapper(address.NewAddressValidationWithCacheWrapper(conf.AddressValidationAPI, conf.MaxCachedItems), conf.ZipcodeToCityStateMapper)

	// Patient/Doctor: Push notification APIs
	mux.Handle(NotificationTokenURLPath, notify.NewNotificationHandler(conf.DataAPI, conf.NotifyConfigs, conf.SNSClient))
	mux.Handle(NotificationPromptStatusURLPath, notify.NewPromptStatusHandler(conf.DataAPI))

	// Patient: Account related APIs
	mux.Handle(PatientSignupURLPath, patient.NewSignupHandler(conf.DataAPI, conf.AuthAPI, addressValidationWithCacheAndHack))
	mux.Handle(PatientInfoURLPath, patient.NewUpdateHandler(conf.DataAPI))
	mux.Handle(PatientAddressURLPath, patient.NewAddressHandler(conf.DataAPI, patient.BILLING_ADDRESS_TYPE))
	mux.Handle(PatientPharmacyURLPath, patient.NewPharmacyHandler(conf.DataAPI))
	mux.Handle(PatientAlertsURLPath, patient_file.NewAlertsHandler(conf.DataAPI))
	mux.Handle(PatientIsAuthenticatedURLPath, handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	mux.Handle(PatientCardURLPath, patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidationAPI))
	mux.Handle(PatientDefaultCardURLPath, patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidationAPI))
	mux.Handle(PatientAuthenticateURLPath, patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.StaticContentURL))
	mux.Handle(LogoutURLPath, patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.StaticContentURL))
	mux.Handle(PatientPCPURLPath, patient.NewPCPHandler(conf.DataAPI))
	mux.Handle(PatientEmergencyContactsURLPath, patient.NewEmergencyContactsHandler(conf.DataAPI))

	// Patient: Patient Case Related APIs
	mux.Handle(CheckEligibilityURLPath, patient.NewCheckCareProvidingEligibilityHandler(conf.DataAPI, addressValidationWithCacheAndHack))
	mux.Handle(PatientVisitURLPath, patient_visit.NewPatientVisitHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle(PatientVisitIntakeURLPath, patient_visit.NewAnswerIntakeHandler(conf.DataAPI))
	mux.Handle(PatientVisitPhotoAnswerURLPath, patient_visit.NewPhotoAnswerIntakeHandler(conf.DataAPI))
	mux.Handle(PatientTreatmentsURLPath, treatment_plan.NewTreatmentsHandler(conf.DataAPI))

	mux.Handle(TreatmentPlanURLPath, treatment_plan.NewTreatmentPlanHandler(conf.DataAPI))
	mux.Handle(TreatmentGuideURLPath, treatment_plan.NewTreatmentGuideHandler(conf.DataAPI))
	mux.Handle(AutocompleteURLPath, handlers.NewAutocompleteHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(PharmacySearchURLPath, patient.NewPharmacySearchHandler(conf.DataAPI, conf.PharmacySearchAPI))

	// Patient: Home API
	mux.Handle(PatientHomeURLPath, patient_case.NewHomeHandler(conf.DataAPI, conf.AuthAPI, addressValidationWithCacheAndHack))

	//Patient/Doctor: Case APIs
	mux.Handle(PatientCasesListURLPath, patient_case.NewListHandler(conf.DataAPI))
	mux.Handle(PatientCasesURLPath, patient_case.NewCaseInfoHandler(conf.DataAPI))
	// Patient: Case APIs
	mux.Handle(PatientCaseNotificationsURLPath, patient_case.NewNotificationsListHandler(conf.DataAPI))

	// Patient/Doctor: Resource guide APIs
	mux.Handle(ResourceGuideURLPath, reslib.NewHandler(conf.DataAPI))
	mux.Handle(ResourceGuidesListURLPath, reslib.NewListHandler(conf.DataAPI))

	// Patient/Doctor: Message APIs
	mux.Handle(CaseMessagesURLPath, messages.NewHandler(conf.DataAPI))
	mux.Handle(CaseMessagesListURLPath, messages.NewListHandler(conf.DataAPI))

	// Doctor: Account APIs
	mux.Handle(DoctorSignupURLPath, doctor.NewSignupDoctorHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle(DoctorAuthenticateURLPath, doctor.NewDoctorAuthenticationHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle(DoctorIsAuthenticatedURLPath, handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	mux.Handle(DoctorQueueURLPath, doctor_queue.NewQueueHandler(conf.DataAPI))

	// Doctor: Prescription related APIs
	mux.Handle(DoctorRXErrorURLPath, doctor.NewPrescriptionErrorHandler(conf.DataAPI))
	mux.Handle(DoctorRXErrorResolveURLPath, doctor.NewPrescriptionErrorIgnoreHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(DoctorRefillRxURLPath, doctor.NewRefillRxHandler(conf.DataAPI, conf.ERxAPI, conf.ERxStatusQueue))
	mux.Handle(DoctorRefillRxDenialReasonsURLPath, doctor.NewRefillRxDenialReasonsHandler(conf.DataAPI))
	mux.Handle(DoctorFTPURLPath, doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(conf.DataAPI))
	mux.Handle(DoctorTreatmentTemplatesURLPath, doctor_treatment_plan.NewTreatmentTemplatesHandler(conf.DataAPI))

	// Doctor: Patient file APIs
	mux.Handle(DoctorPatientTreatmentsURLPath, patient_file.NewDoctorPatientTreatmentsHandler(conf.DataAPI))
	mux.Handle(DoctorPatientInfoURLPath, patient_file.NewDoctorPatientHandler(conf.DataAPI, conf.ERxAPI, conf.AddressValidationAPI))
	mux.Handle(DoctorPatientVisitsURLPath, patient_file.NewPatientVisitsHandler(conf.DataAPI))
	mux.Handle(DoctorPatientPharmacyURLPath, patient_file.NewDoctorUpdatePatientPharmacyHandler(conf.DataAPI))
	mux.Handle(DoctorTreatmentPlansURLPath, doctor_treatment_plan.NewDoctorTreatmentPlanHandler(conf.DataAPI, conf.ERxAPI, conf.ERxStatusQueue, conf.ERxRouting))
	mux.Handle(DoctorTreatmentPlansListURLPath, doctor_treatment_plan.NewListHandler(conf.DataAPI))
	mux.Handle(DoctorPharmacySearchURLPath, doctor.NewPharmacySearchHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(DoctorVisitReviewURLPath, patient_file.NewDoctorPatientVisitReviewHandler(conf.DataAPI))
	mux.Handle(DoctorVisitDiagnosisURLPath, patient_visit.NewDiagnosePatientHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle(DoctorSelectMedicationURLPath, doctor_treatment_plan.NewMedicationSelectHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(DoctorVisitTreatmentsURLPath, doctor_treatment_plan.NewTreatmentsHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(DoctorMedicationSearchURLPath, handlers.NewAutocompleteHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(DoctorMedicationStrengthsURLPath, doctor_treatment_plan.NewMedicationStrengthSearchHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(DoctorMedicationDispenseUnitsURLPath, doctor_treatment_plan.NewMedicationDispenseUnitsHandler(conf.DataAPI))
	mux.Handle(DoctorRegimenURLPath, doctor_treatment_plan.NewRegimenHandler(conf.DataAPI))
	mux.Handle(DoctorAdviceURLPath, doctor_treatment_plan.NewAdviceHandler(conf.DataAPI))
	mux.Handle(DoctorSavedMessagesURLPath, doctor_treatment_plan.NewSavedMessageHandler(conf.DataAPI))
	mux.Handle(DoctorCaseClaimURLPath, doctor_queue.NewClaimPatientCaseAccessHandler(conf.DataAPI, conf.MetricsRegistry.Scope("doctor_queue")))
	mux.Handle(DoctorAssignCaseURLPath, messages.NewAssignHandler(conf.DataAPI))
	mux.Handle(DoctorCaseCareTeamURLPath, patient_case.NewCareTeamHandler(conf.DataAPI))

	// Miscellaneous APIs
	mux.Handle(ContentURLPath, handlers.NewStaticContentHandler(conf.DataAPI, conf.CloudStorageAPI, conf.ContentBucket, conf.AWSRegion))
	mux.Handle(PingURLPath, handlers.NewPingHandler())
	mux.Handle(PhotoURLPath, photos.NewHandler(conf.DataAPI, conf.Stores["photos"]))
	mux.Handle(LayoutUploadURLPath, layout.NewLayoutUploadHandler(conf.DataAPI))
	mux.Handle(AppEventURLPath, app_event.NewHandler())
	mux.Handle(AnalyticsURLPath, analytics.NewHandler(conf.AnalyticsLogger, conf.MetricsRegistry.Scope("analytics.event.client")))
	mux.Handle(ResetPasswordURLPath, passreset.NewForgotPasswordHandler(conf.DataAPI, conf.AuthAPI, conf.EmailService, conf.CustomerSupportEmail, conf.WebSubdomain))
	mux.Handle(CareProviderProfileURLPath, handlers.NewCareProviderProfileHandler(conf.DataAPI))
	// add the api to create demo visits to every environment except production
	if !environment.IsProd() {
		mux.Handle("/v1/doctor/demo/patient_visit", demo.NewHandler(conf.DataAPI, conf.CloudStorageAPI, conf.AWSRegion))
		mux.Handle("/v1/doctor/demo/favorite_treatment_plan", demo.NewFavoriteTreatmentPlanHandler(conf.DataAPI))
	}

	return mux
}
