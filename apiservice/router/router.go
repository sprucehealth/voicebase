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

type RouterConfig struct {
	DataAPI               api.DataAPI
	AuthAPI               api.AuthAPI
	AddressValidationAPI  address.AddressValidationAPI
	PharmacySearchAPI     pharmacy.PharmacySearchAPI
	SNSClient             *sns.SNS
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
	mux.Handle("/v1/notification/token", notify.NewNotificationHandler(conf.DataAPI, conf.NotifyConfigs, conf.SNSClient))
	mux.Handle("/v1/notification/prompt_status", notify.NewPromptStatusHandler(conf.DataAPI))

	// Patient: Account related APIs
	mux.Handle("/v1/patient", patient.NewSignupHandler(conf.DataAPI, conf.AuthAPI, conf.AddressValidationAPI))
	mux.Handle("/v1/patient/info", patient.NewUpdateHandler(conf.DataAPI))
	mux.Handle("/v1/patient/address/billing", patient.NewAddressHandler(conf.DataAPI, patient.BILLING_ADDRESS_TYPE))
	mux.Handle("/v1/patient/pharmacy", patient.NewPharmacyHandler(conf.DataAPI))
	mux.Handle("/v1/patient/alerts", patient_file.NewAlertsHandler(conf.DataAPI))
	mux.Handle("/v1/patient/isauthenticated", handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	mux.Handle("/v1/reset_password", passreset.NewForgotPasswordHandler(conf.DataAPI, conf.AuthAPI, conf.EmailService, conf.CustomerSupportEmail, conf.WebSubdomain))
	mux.Handle("/v1/credit_card", patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidationAPI))
	mux.Handle("/v1/credit_card/default", patient.NewCardsHandler(conf.DataAPI, conf.PaymentAPI, conf.AddressValidationAPI))
	mux.Handle("/v1/authenticate", patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.StaticContentURL))
	mux.Handle("/v1/logout", patient.NewAuthenticationHandler(conf.DataAPI, conf.AuthAPI, conf.StaticContentURL))

	// Patient: Patient Case Related APIs
	mux.Handle("/v1/check_eligibility", patient.NewCheckCareProvidingEligibilityHandler(conf.DataAPI, conf.AddressValidationAPI))
	mux.Handle("/v1/patient/visit", patient_visit.NewPatientVisitHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle("/v1/patient/visit/answer", patient_visit.NewAnswerIntakeHandler(conf.DataAPI))
	mux.Handle("/v1/patient/visit/photo_answer", patient_visit.NewPhotoAnswerIntakeHandler(conf.DataAPI))
	mux.Handle("/v1/patient/treatments", treatment_plan.NewTreatmentsHandler(conf.DataAPI))

	mux.Handle("/v1/treatment_plan", treatment_plan.NewTreatmentPlanHandler(conf.DataAPI))
	mux.Handle("/v1/treatment_guide", treatment_plan.NewTreatmentGuideHandler(conf.DataAPI))
	mux.Handle("/v1/autocomplete", handlers.NewAutocompleteHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle("/v1/pharmacy_search", patient.NewPharmacySearchHandler(conf.DataAPI, conf.PharmacySearchAPI))

	// Patient: Home API
	mux.Handle("/v1/patient/home", patient_case.NewHomeHandler(conf.DataAPI, conf.AuthAPI))

	//Patient/Doctor: Case APIs
	mux.Handle("/v1/cases/list", patient_case.NewListHandler(conf.DataAPI))
	mux.Handle("/v1/cases", patient_case.NewCaseInfoHandler(conf.DataAPI))
	// Patient: Case APIs
	mux.Handle("/v1/patient/case/notifications", patient_case.NewNotificationsListHandler(conf.DataAPI))

	// Patient/Doctor: Resource guide APIs
	mux.Handle("/v1/resourceguide", reslib.NewHandler(conf.DataAPI))
	mux.Handle("/v1/resourceguide/list", reslib.NewListHandler(conf.DataAPI))

	// Patient/Doctor: Message APIs
	mux.Handle("/v1/case/messages", messages.NewHandler(conf.DataAPI))
	mux.Handle("/v1/case/messages/list", messages.NewListHandler(conf.DataAPI))

	// Doctor: Account APIs
	mux.Handle("/v1/doctor/signup", doctor.NewSignupDoctorHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle("/v1/doctor/authenticate", doctor.NewDoctorAuthenticationHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle("/v1/doctor/isauthenticated", handlers.NewIsAuthenticatedHandler(conf.AuthAPI))
	mux.Handle("/v1/doctor/queue", doctor_queue.NewQueueHandler(conf.DataAPI))

	// Doctor: Prescription related APIs
	mux.Handle("/v1/doctor/rx/error", doctor.NewPrescriptionErrorHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/rx/error/resolve", doctor.NewPrescriptionErrorIgnoreHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle("/v1/doctor/rx/refill/request", doctor.NewRefillRxHandler(conf.DataAPI, conf.ERxAPI, conf.ERxStatusQueue))
	mux.Handle("/v1/doctor/rx/refill/denial_reasons", doctor.NewRefillRxDenialReasonsHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/favorite_treatment_plans", doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/treatment/templates", doctor_treatment_plan.NewTreatmentTemplatesHandler(conf.DataAPI))

	// Doctor: Patient file APIs
	mux.Handle("/v1/doctor/patient/treatments", patient_file.NewDoctorPatientTreatmentsHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/patient", patient_file.NewDoctorPatientHandler(conf.DataAPI, conf.ERxAPI, conf.AddressValidationAPI))
	mux.Handle("/v1/doctor/patient/visits", patient_file.NewPatientVisitsHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/patient/pharmacy", patient_file.NewDoctorUpdatePatientPharmacyHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/treatment_plans", doctor_treatment_plan.NewDoctorTreatmentPlanHandler(conf.DataAPI, conf.ERxAPI, conf.ERxStatusQueue, conf.ERxRouting))
	mux.Handle("/v1/doctor/treatment_plans/list", doctor_treatment_plan.NewListHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/pharmacy", doctor.NewPharmacySearchHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle("/v1/doctor/visit/review", patient_file.NewDoctorPatientVisitReviewHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/visit/diagnosis", patient_visit.NewDiagnosePatientHandler(conf.DataAPI, conf.AuthAPI))
	mux.Handle("/v1/doctor/visit/treatment/new", doctor_treatment_plan.NewMedicationSelectHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle("/v1/doctor/visit/treatment/treatments", doctor_treatment_plan.NewTreatmentsHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle("/v1/doctor/visit/treatment/medication_suggestions", handlers.NewAutocompleteHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle("/v1/doctor/visit/treatment/medication_strengths", doctor_treatment_plan.NewMedicationStrengthSearchHandler(conf.DataAPI, conf.ERxAPI))
	mux.Handle("/v1/doctor/visit/treatment/medication_dispense_units", doctor_treatment_plan.NewMedicationDispenseUnitsHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/visit/regimen", doctor_treatment_plan.NewRegimenHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/visit/advice", doctor_treatment_plan.NewAdviceHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/saved_messages", doctor_treatment_plan.NewSavedMessageHandler(conf.DataAPI))
	mux.Handle("/v1/doctor/patient/case/claim", doctor_queue.NewClaimPatientCaseAccessHandler(conf.DataAPI, conf.MetricsRegistry.Scope("doctor_queue")))

	// Miscellaneous APIs
	mux.Handle("/v1/content", handlers.NewStaticContentHandler(conf.DataAPI, conf.CloudStorageAPI, conf.ContentBucket, conf.AWSRegion))
	mux.Handle("/v1/ping", handlers.NewPingHandler())
	mux.Handle("/v1/photo", photos.NewHandler(conf.DataAPI, conf.Stores["photos"]))
	mux.Handle("/v1/layouts/upload", layout.NewLayoutUploadHandler(conf.DataAPI))
	mux.Handle("/v1/app_event", app_event.NewHandler())
	mux.Handle("/v1/event/client", analytics.NewHandler(conf.AnalyticsLogger, conf.MetricsRegistry.Scope("analytics.event.client")))

	// add the api to create demo visits to every environment except production
	if !environment.IsProd() {
		mux.Handle("/v1/doctor/demo/patient_visit", demo.NewHandler(conf.DataAPI, conf.CloudStorageAPI, conf.AWSRegion))
		mux.Handle("/v1/doctor/demo/favorite_treatment_plan", demo.NewFavoriteTreatmentPlanHandler(conf.DataAPI))
	}

	return mux
}
