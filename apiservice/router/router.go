package router

import (
	"net/http"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/common/handlers"
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
	NotificationTokenURLPath             = "/v1/notification/token"
	NotificationPromptStatusURLPath      = "/v1/notification/prompt_status"
	PatientSignupURLPath                 = "/v1/patient"
	PatientInfoURLPath                   = "/v1/patient/info"
	PatientAddressURLPath                = "/v1/patient/address/billing"
	PatientPharmacyURLPath               = "/v1/patient/pharmacy"
	PatientAlertsURLPath                 = "/v1/patient/alerts"
	PatientIsAuthenticatedURLPath        = "/v1/patient/isauthenticated"
	PatientCardURLPath                   = "/v1/credit_card"
	PatientDefaultCardURLPath            = "/v1/credit_card/default"
	PatientAuthenticateURLPath           = "/v1/authenticate"
	PatientVisitURLPath                  = "/v1/patient/visit"
	PatientVisitIntakeURLPath            = "/v1/patient/visit/answer"
	PatientVisitPhotoAnswerURLPath       = "/v1/patient/visit/photo_answer"
	PatientTreatmentsURLPath             = "/v1/patient/treatments"
	PatientHomeURLPath                   = "/v1/patient/home"
	PatientCasesListURLPath              = "/v1/cases/list"
	PatientCasesURLPath                  = "/v1/cases"
	PatientCaseNotificationsURLPath      = "/v1/patient/case/notifications"
	TreatmentPlanURLPath                 = "/v1/treatment_plan"
	TreatmentGuideURLPath                = "/v1/treatment_guide"
	AutocompleteURLPath                  = "/v1/autocomplete"
	PharmacySearchURLPath                = "/v1/pharmacy_search"
	CheckEligibilityURLPath              = "/v1/check_eligibility"
	LogoutURLPath                        = "/v1/logout"
	ResetPasswordURLPath                 = "/v1/reset_password"
	ResourceGuideURLPath                 = "/v1/resourceguide"
	ResourceGuidesListURLPath            = "/v1/resourceguide/list"
	CaseMessagesURLPath                  = "/v1/case/messages"
	CaseMessagesListURLPath              = "/v1/case/messages/list"
	DoctorSignupURLPath                  = "/v1/doctor/signup"
	DoctorAuthenticateURLPath            = "/v1/doctor/authenticate"
	DoctorIsAuthenticatedURLPath         = "/v1/doctor/isauthenticated"
	DoctorQueueURLPath                   = "/v1/doctor/queue"
	DoctorRXErrorURLPath                 = "/v1/doctor/rx/error"
	DoctorRXErrorResolveURLPath          = "/v1/doctor/rx/error/resolve"
	DoctorRefillRxURLPath                = "/v1/doctor/rx/refill/request"
	DoctorRefillRxDenialReasonsURLPath   = "/v1/doctor/rx/refill/denial_reasons"
	DoctorFTPURLPath                     = "/v1/doctor/favorite_treatment_plans"
	DoctorTreatmentTemplatesURLPath      = "/v1/doctor/treatment/templates"
	DoctorPatientTreatmentsURLPath       = "/v1/doctor/patient/treatments"
	DoctorPatientInfoURLPath             = "/v1/doctor/patient"
	DoctorPatientVisitsURLPath           = "/v1/doctor/patient/visits"
	DoctorPatientPharmacyURLPath         = "/v1/doctor/patient/pharmacy"
	DoctorTreatmentPlansURLPath          = "/v1/doctor/treatment_plans"
	DoctorTreatmentPlansListURLPath      = "/v1/doctor/treatment_plans/list"
	DoctorPharmacySearchURLPath          = "/v1/doctor/pharmacy"
	DoctorVisitReviewURLPath             = "/v1/doctor/visit/review"
	DoctorVisitDiagnosisURLPath          = "/v1/doctor/visit/diagnosis"
	DoctorSelectMedicationURLPath        = "/v1/doctor/visit/treatment/new"
	DoctorVisitTreatmentsURLPath         = "/v1/doctor/visit/treatment/treatments"
	DoctorMedicationSearchURLPath        = "/v1/doctor/visit/treatment/medication_suggestions"
	DoctorMedicationStrengthsURLPath     = "/v1/doctor/visit/treatment/medication_strengths"
	DoctorMedicationDispenseUnitsURLPath = "/v1/doctor/visit/treatment/medication_dispense_units"
	DoctorRegimenURLPath                 = "/v1/doctor/visit/regimen"
	DoctorAdviceURLPath                  = "/v1/doctor/visit/advice"
	DoctorSavedMessagesURLPath           = "/v1/doctor/saved_messages"
	DoctorCaseClaimURLPath               = "/v1/doctor/patient/case/claim"
	ContentURLPath                       = "/v1/content"
	PingURLPath                          = "/v1/ping"
	PhotoURLPath                         = "/v1/photo"
	LayoutUploadURLPath                  = "/v1/layous/upload"
	AppEventURLPath                      = "/v1/app_event"
	AnalyticsURLPath                     = "/v1/event/client"
)

type RouterConfig struct {
	DataAPI               api.DataAPI
	AuthAPI               api.AuthAPI
	AddressValidationAPI  address.AddressValidationAPI
	PharmacySearchAPI     pharmacy.PharmacySearchAPI
	SNSClient             sns.SNSService
	PaymentAPI            payment.PaymentAPI
	NotifyConfigs         *config.NotificationConfigs
	NotificationManager   *notify.NotificationManager
	ERxStatusQueue        *common.SQSQueue
	ERxAPI                erx.ERxAPI
	EmailService          email.Service
	MetricsRegistry       metrics.Registry
	TwilioClient          *twilio.Client
	CloudStorageAPI       api.CloudStorageAPI
	Stores                map[string]storage.Store
	AnalyticsLogger       analytics.Logger
	ERxRouting            bool
	JBCQMinutesThreshold  int
	CustomerSupportEmail  string
	TechnicalSupportEmail string
	APISubdomain          string
	WebSubdomain          string
	StaticContentURL      string
	ContentBucket         string
	AWSRegion             string
}

func New(conf *RouterConfig) http.Handler {

	// Initialize listeneres
	doctor_queue.InitListeners(conf.DataAPI, conf.NotificationManager, conf.MetricsRegistry.Scope("doctor_queue"), conf.JBCQMinutesThreshold)
	doctor_treatment_plan.InitListeners(conf.DataAPI)
	notify.InitListeners(conf.DataAPI)
	support.InitListeners(conf.TechnicalSupportEmail, conf.CustomerSupportEmail, conf.NotificationManager)
	patient_case.InitListeners(conf.DataAPI, conf.NotificationManager)
	patient_visit.InitListeners(conf.DataAPI)
	demo.InitListeners(conf.DataAPI, conf.APISubdomain)

	// Start worker to check for expired items in the global case queue
	doctor_queue.StartClaimedItemsExpirationChecker(conf.DataAPI, conf.MetricsRegistry.Scope("doctor_queue"))
	if conf.ERxRouting {
		app_worker.StartWorkerToUpdatePrescriptionStatusForPatient(conf.DataAPI, conf.ERxAPI, conf.ERxStatusQueue, conf.MetricsRegistry.Scope("check_erx_status"))
		app_worker.StartWorkerToCheckForRefillRequests(conf.DataAPI, conf.ERxAPI, conf.MetricsRegistry.Scope("check_rx_refill_requests"))
		app_worker.StartWorkerToCheckRxErrors(dataApi, doseSpotService, metricsRegistry.Scope("check_rx_errors"))
	}

	mux := apiservice.NewAuthServeMux(conf.AuthAPI, conf.MetricsRegistry.Scope("restapi"))

	// Patient/Doctor: Push notification APIs
	mux.Handle(NotificationTokenURLPath, notify.NewNotificationHandler(conf.DataAPI, conf.NotifyConfigs, conf.SNSClient))
	mux.Handle(NotificationPromptStatusURLPath, notify.NewPromptStatusHandler(conf.DataAPI))

	// Patient: Account related APIs
	mux.Handle(PatientSignupURLPath, patient.NewSignupHandler(conf.DataAPI, conf.AuthAPI, conf.AddressValidationAPI))
	mux.Handle(PatientInfoURLPath, patient.NewUpdateHandler(conf.DataAPI))
	mux.Handle(PatientAddressURLPath, patient.NewAddressHandler(conf.DataAPI, patient.BILLING_ADDRESS_TYPE))
	mux.Handle(PatientPharmacyURLPath, patient.NewPharmacyHandler(conf.DataAPI))
	mux.Handle(PatientAlertsURLPath, patient_file.NewAlertsHandler(conf.DataAPI))
	mux.Handle(PatientIsAuthenticatedURLPath, handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	mux.Handle(PatientCardURLPath, patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidationAPI))
	mux.Handle(PatientDefaultCardURLPath, patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidationAPI))
	mux.Handle(PatientAuthenticateURLPath, patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.StaticContentURL))
	mux.Handle(LogoutURLPath, patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.StaticContentURL))

	// Patient: Patient Case Related APIs
	mux.Handle(CheckEligibilityURLPath, patient.NewCheckCareProvidingEligibilityHandler(conf.DataAPI, conf.AddressValidationAPI))
	mux.Handle(PatientVisitURLPath, patient_visit.NewPatientVisitHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle(PatientVisitIntakeURLPath, patient_visit.NewAnswerIntakeHandler(conf.DataAPI))
	mux.Handle(PatientVisitPhotoAnswerURLPath, patient_visit.NewPhotoAnswerIntakeHandler(conf.DataAPI))
	mux.Handle(PatientTreatmentsURLPath, treatment_plan.NewTreatmentsHandler(conf.DataAPI))

	mux.Handle(TreatmentPlanURLPath, treatment_plan.NewTreatmentPlanHandler(conf.DataAPI))
	mux.Handle(TreatmentGuideURLPath, treatment_plan.NewTreatmentGuideHandler(conf.DataAPI))
	mux.Handle(AutocompleteURLPath, handlers.NewAutocompleteHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle(PharmacySearchURLPath, patient.NewPharmacySearchHandler(conf.DataAPI, conf.PharmacySearchAPI))

	// Patient: Home API
	mux.Handle(PatientHomeURLPath, patient_case.NewHomeHandler(conf.DataAPI, conf.AuthAPI))

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

	// Miscellaneous APIs
	mux.Handle(ContentURLPath, handlers.NewStaticContentHandler(conf.DataAPI, conf.CloudStorageAPI, conf.ContentBucket, conf.AWSRegion))
	mux.Handle(PingURLPath, handlers.NewPingHandler())
	mux.Handle(PhotoURLPath, photos.NewHandler(conf.DataAPI, conf.Stores["photos"]))
	mux.Handle(LayoutUploadURLPath, layout.NewLayoutUploadHandler(conf.DataAPI))
	mux.Handle(AppEventURLPath, app_event.NewHandler())
	mux.Handle(AnalyticsURLPath, analytics.NewHandler(conf.AnalyticsLogger, conf.MetricsRegistry.Scope("analytics.event.client")))
	mux.Handle(ResetPasswordURLPath, passreset.NewForgotPasswordHandler(conf.DataAPI, conf.AuthAPI, conf.EmailService, conf.CustomerSupportEmail, conf.WebSubdomain))

	// add the api to create demo visits to every environment except production
	if !environment.IsProd() {
		mux.Handle("/v1/doctor/demo/patient_visit", demo.NewHandler(conf.DataAPI, conf.CloudStorageAPI, conf.AWSRegion))
		mux.Handle("/v1/doctor/demo/favorite_treatment_plan", demo.NewFavoriteTreatmentPlanHandler(conf.DataAPI))
	}

	return mux
}
