package main

import (
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/httputil"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/demo"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/layout"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/maps"
	"github.com/sprucehealth/backend/libs/payment/stripe"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/passreset"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_case"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/photos"
	"github.com/sprucehealth/backend/reslib"
	"github.com/sprucehealth/backend/support"
	"github.com/sprucehealth/backend/surescripts/pharmacy"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/treatment_plan"
	"github.com/sprucehealth/backend/www/router"
)

const (
	defaultMaxInMemoryPhotoMB = 2
)

func connectDB(conf *Config) *sql.DB {
	db, err := conf.DB.Connect(conf.BaseConfig)
	if err != nil {
		log.Fatal(err)
	}

	if num, err := strconv.Atoi(config.MigrationNumber); err == nil {
		var latestMigration int
		if err := db.QueryRow("SELECT MAX(migration_id) FROM migrations").Scan(&latestMigration); err != nil {
			if !conf.Debug {
				log.Fatalf("Failed to query for latest migration: %s", err.Error())
			} else {
				log.Printf("Failed to query for latest migration: %s", err.Error())
			}
		}
		if latestMigration != num {
			if conf.Debug {
				golog.Warningf("Current database migration = %d, want %d", latestMigration, num)
			} else {
				// TODO: eventually make this Fatal once everything has been fully tested
				golog.Errorf("Current database migration = %d, want %d", latestMigration, num)
			}
		}
	} else if !conf.Debug {
		// TODO: eventually make this Fatal once everything has been fully tested
		golog.Errorf("MigrationNumber not set and not debug")
	}

	return db
}

func main() {
	conf := DefaultConfig
	_, err := config.Parse(&conf)
	if err != nil {
		log.Fatal(err)
	}

	if conf.Debug {
		golog.Default().SetLevel(golog.DEBUG)
	} else if conf.Environment == "dev" {
		golog.Default().SetLevel(golog.INFO)
	}

	conf.Validate()

	awsAuth, err := conf.AWSAuth()
	if err != nil {
		log.Fatalf("Failed to get AWS auth: %+v", err)
	}
	stores := make(map[string]storage.Store)
	for name, c := range conf.Storage {
		switch strings.ToLower(c.Type) {
		default:
			log.Fatalf("Unknown storage type %s for name %s", c.Type, name)
		case "s3":
			stores[name] = storage.NewS3(awsAuth, c.Region, c.Bucket, c.Prefix)
		}
	}

	db := connectDB(&conf)
	defer db.Close()

	dataApi, err := api.NewDataService(db)
	if err != nil {
		log.Fatalf("Unable to initialize data service layer: %s", err)
	}

	metricsRegistry := metrics.NewRegistry()
	conf.StartReporters(metricsRegistry)

	if conf.InfoAddr != "" {
		http.Handle("/metrics", metrics.RegistryHandler(metricsRegistry))
		go func() {
			log.Fatal(http.ListenAndServe(conf.InfoAddr, nil))
		}()
	}

	authAPI := &api.Auth{
		DB:             db,
		ExpireDuration: time.Duration(conf.AuthTokenExpiration) * time.Second,
		RenewDuration:  time.Duration(conf.AuthTokenRenew) * time.Second,
		Hasher:         api.NewBcryptHasher(0),
	}

	sigKeys := make([][]byte, len(conf.SecretSignatureKeys))
	for i, k := range conf.SecretSignatureKeys {
		// No reason to decode the keys to binary. They'll be slightly longer
		// as ascii but include no less entropy.
		sigKeys[i] = []byte(k)
	}
	signer := &common.Signer{
		Keys: sigKeys,
	}

	restAPIMux := buildRESTAPI(&conf, dataApi, authAPI, stores, metricsRegistry)
	webMux := buildWWW(&conf, dataApi, authAPI, signer, stores, metricsRegistry)

	router := mux.NewRouter()
	router.Host(conf.APISubdomain + ".{domain:.+}").Handler(restAPIMux)
	router.Host(conf.WebSubdomain + ".{domain:.+}").Handler(webMux)

	conf.SetupLogging()

	serve(&conf, router)
}

func buildWWW(conf *Config, dataApi api.DataAPI, authAPI api.AuthAPI, signer *common.Signer, stores map[string]storage.Store, metricsRegistry metrics.Registry) http.Handler {
	twilioCli, err := conf.Twilio.Client()
	if err != nil {
		if conf.Debug {
			log.Println(err.Error())
		} else {
			log.Fatal(err.Error())
		}
	}

	stripeCli := &stripe.StripeService{
		SecretKey:      conf.StripeSecretKey,
		PublishableKey: conf.StripePublishableKey,
	}

	return router.New(dataApi, authAPI, twilioCli, conf.Twilio.FromNumber,
		email.NewService(conf.Email, metricsRegistry.Scope("email")), conf.Support.CustomerSupportEmail,
		conf.WebSubdomain, stripeCli, signer, stores, metricsRegistry.Scope("www"))
}

func buildRESTAPI(conf *Config, dataApi api.DataAPI, authAPI api.AuthAPI, stores map[string]storage.Store, metricsRegistry metrics.Registry) http.Handler {
	twilioCli, err := conf.Twilio.Client()
	if err != nil {
		if conf.Debug {
			log.Println(err.Error())
		} else {
			log.Fatal(err.Error())
		}
	}

	awsAuth, err := conf.AWSAuth()
	if err != nil {
		log.Fatalf("Failed to get AWS auth: %+v", err)
	}

	emailService := email.NewService(conf.Email, metricsRegistry.Scope("email"))

	surescriptsPharmacySearch, err := pharmacy.NewSurescriptsPharmacySearch(conf.PharmacyDB, conf.Environment)
	if err != nil {
		if conf.Debug {
			log.Printf("Unable to initialize pharmacy search: %s", err)
		} else {
			log.Fatalf("Unable to initialize pharmacy search: %s", err)
		}
	}

	var erxStatusQueue *common.SQSQueue
	if conf.ERxQueue != "" {
		var err error
		erxStatusQueue, err = common.NewQueue(awsAuth, aws.Regions[conf.AWSRegion], conf.ERxQueue)
		if err != nil {
			log.Fatal("Unable to get erx queue for sending prescriptions to: " + err.Error())
		}
	} else if conf.ERxRouting {
		log.Fatal("ERxQueue not configured but ERxRouting is enabled")
	}
	snsClient := &sns.SNS{
		Region: aws.USEast,
		Client: &aws.Client{
			Auth: awsAuth,
		},
	}
	smartyStreetsService := &address.SmartyStreetsService{
		AuthId:    conf.SmartyStreets.AuthId,
		AuthToken: conf.SmartyStreets.AuthToken,
	}

	mapsService := maps.NewGoogleMapsService(metricsRegistry.Scope("google_maps_api"))
	doseSpotService := erx.NewDoseSpotService(conf.DoseSpot.ClinicId, conf.DoseSpot.UserId, conf.DoseSpot.ClinicKey, conf.DoseSpot.SOAPEndpoint, conf.DoseSpot.APIEndpoint, metricsRegistry.Scope("dosespot_api"))
	autocompleteHandler := apiservice.NewAutocompleteHandler(dataApi, doseSpotService)

	notificationManager := notify.NewManager(dataApi, snsClient, twilioCli, emailService,
		conf.Twilio.FromNumber, conf.AlertEmail, conf.NotifiyConfigs, metricsRegistry.Scope("notify"))

	// Initialize listeneres
	doctor_queue.InitListeners(dataApi, notificationManager, metricsRegistry.Scope("doctor_queue"))
	doctor_treatment_plan.InitListeners(dataApi)
	notify.InitListeners(dataApi)
	support.InitListeners(conf.Support.TechnicalSupportEmail, conf.Support.CustomerSupportEmail, notificationManager)
	patient_case.InitListeners(dataApi, notificationManager)
	patient_visit.InitListeners(dataApi)
	demo.InitListeners(dataApi, conf.WebSubdomain)
	// Start worker to check for expired items in the global case queue
	doctor_queue.StartClaimedItemsExpirationChecker(dataApi, metricsRegistry.Scope("doctor_queue"))

	cloudStorageApi := api.NewCloudStorageService(awsAuth)
	checkElligibilityHandler := &apiservice.CheckCareProvidingElligibilityHandler{DataApi: dataApi, AddressValidationApi: smartyStreetsService, StaticContentUrl: conf.StaticContentBaseUrl}
	updatePatientBillingAddress := &apiservice.UpdatePatientAddressHandler{DataApi: dataApi, AddressType: apiservice.BILLING_ADDRESS_TYPE}
	medicationStrengthSearchHandler := &apiservice.MedicationStrengthSearchHandler{DataApi: dataApi, ERxApi: doseSpotService}
	newTreatmentHandler := &apiservice.NewTreatmentHandler{DataApi: dataApi, ERxApi: doseSpotService}
	medicationDispenseUnitHandler := &apiservice.MedicationDispenseUnitsHandler{DataApi: dataApi}
	pharmacySearchHandler := &apiservice.PharmacyTextSearchHandler{PharmacySearchService: surescriptsPharmacySearch, DataApi: dataApi, MapsService: mapsService}
	pingHandler := apiservice.PingHandler(0)

	staticContentHandler := &apiservice.StaticContentHandler{
		DataApi:               dataApi,
		ContentStorageService: cloudStorageApi,
		BucketLocation:        conf.ContentBucket,
		Region:                conf.AWSRegion,
	}

	doctorPrescriptionErrorHandler := &apiservice.DoctorPrescriptionErrorHandler{
		DataApi: dataApi,
	}

	doctorPrescriptionErrorIgnoreHandler := &apiservice.DoctorPrescriptionErrorIgnoreHandler{
		DataApi: dataApi,
		ErxApi:  doseSpotService,
	}

	doctorRefillRequestHandler := &apiservice.DoctorRefillRequestHandler{
		DataApi:        dataApi,
		ErxApi:         doseSpotService,
		ErxStatusQueue: erxStatusQueue,
	}

	refillRequestDenialReasonsHandler := &apiservice.RefillRequestDenialReasonsHandler{
		DataApi: dataApi,
	}

	patientCardsHandler := &apiservice.PatientCardsHandler{
		DataApi:              dataApi,
		PaymentApi:           &stripe.StripeService{SecretKey: conf.StripeSecretKey},
		AddressValidationApi: smartyStreetsService,
	}

	doctorPharmacySearchHandler := &apiservice.DoctorPharmacySearchHandler{
		DataApi: dataApi,
		ErxApi:  doseSpotService,
	}

	mux := apiservice.NewAuthServeMux(authAPI, metricsRegistry.Scope("restapi"))

	// Patient/Doctor: Push notification APIs
	mux.Handle("/v1/notification/token", notify.NewNotificationHandler(dataApi, conf.NotifiyConfigs, snsClient))
	mux.Handle("/v1/notification/prompt_status", notify.NewPromptStatusHandler(dataApi))

	// Patient: Account related APIs
	mux.Handle("/v1/patient", patient.NewSignupHandler(dataApi, authAPI, smartyStreetsService))
	mux.Handle("/v1/patient/info", patient.NewUpdateHandler(dataApi))
	mux.Handle("/v1/patient/address/billing", updatePatientBillingAddress)
	mux.Handle("/v1/patient/pharmacy", apiservice.NewUpdatePatientPharmacyHandler(dataApi))
	mux.Handle("/v1/patient/alerts", patient_file.NewAlertsHandler(dataApi))
	mux.Handle("/v1/patient/isauthenticated", apiservice.NewIsAuthenticatedHandler(authAPI))
	mux.Handle("/v1/reset_password", passreset.NewForgotPasswordHandler(dataApi, authAPI, emailService, conf.Support.CustomerSupportEmail, conf.WebSubdomain))
	mux.Handle("/v1/credit_card", patientCardsHandler)
	mux.Handle("/v1/credit_card/default", patientCardsHandler)
	mux.Handle("/v1/authenticate", patient.NewAuthenticationHandler(dataApi, authAPI, conf.StaticContentBaseUrl))
	mux.Handle("/v1/logout", patient.NewAuthenticationHandler(dataApi, authAPI, conf.StaticContentBaseUrl))

	// Patient: Patient Case Related APIs
	mux.Handle("/v1/check_eligibility", checkElligibilityHandler)
	mux.Handle("/v1/patient/visit", patient_visit.NewPatientVisitHandler(dataApi, authAPI))
	mux.Handle("/v1/patient/visit/answer", patient_visit.NewAnswerIntakeHandler(dataApi))
	mux.Handle("/v1/patient/visit/photo_answer", patient_visit.NewPhotoAnswerIntakeHandler(dataApi))
	mux.Handle("/v1/patient/treatments", treatment_plan.NewTreatmentsHandler(dataApi))

	mux.Handle("/v1/treatment_plan", treatment_plan.NewTreatmentPlanHandler(dataApi))
	mux.Handle("/v1/treatment_guide", treatment_plan.NewTreatmentGuideHandler(dataApi))
	mux.Handle("/v1/autocomplete", autocompleteHandler)
	mux.Handle("/v1/pharmacy_search", pharmacySearchHandler)

	// Patient: Home API
	mux.Handle("/v1/patient/home", patient_case.NewHomeHandler(dataApi, authAPI))

	//Patient/Doctor: Case APIs
	mux.Handle("/v1/cases/list", patient_case.NewListHandler(dataApi))
	mux.Handle("/v1/cases", patient_case.NewCaseInfoHandler(dataApi))
	// Patient: Case APIs
	mux.Handle("/v1/patient/case/notifications", patient_case.NewNotificationsListHandler(dataApi))

	// Patient/Doctor: Resource guide APIs
	mux.Handle("/v1/resourceguide", reslib.NewHandler(dataApi))
	mux.Handle("/v1/resourceguide/list", reslib.NewListHandler(dataApi))

	// Patient/Doctor: Message APIs
	mux.Handle("/v1/case/messages", messages.NewHandler(dataApi))
	mux.Handle("/v1/case/messages/list", messages.NewListHandler(dataApi))
	mux.Handle("/v1/case/messages/read", messages.NewReadHandler(dataApi))

	// Doctor: Account APIs
	mux.Handle("/v1/doctor/signup", apiservice.NewSignupDoctorHandler(dataApi, authAPI, conf.Environment))
	mux.Handle("/v1/doctor/authenticate", apiservice.NewDoctorAuthenticationHandler(dataApi, authAPI))
	mux.Handle("/v1/doctor/isauthenticated", apiservice.NewIsAuthenticatedHandler(authAPI))
	mux.Handle("/v1/doctor/queue", doctor_queue.NewQueueHandler(dataApi))

	// Doctor: Prescription related APIs
	mux.Handle("/v1/doctor/rx/error", doctorPrescriptionErrorHandler)
	mux.Handle("/v1/doctor/rx/error/resolve", doctorPrescriptionErrorIgnoreHandler)
	mux.Handle("/v1/doctor/rx/refill/request", doctorRefillRequestHandler)
	mux.Handle("/v1/doctor/rx/refill/denial_reasons", refillRequestDenialReasonsHandler)

	mux.Handle("/v1/doctor/favorite_treatment_plans", doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(dataApi))
	mux.Handle("/v1/doctor/treatment/templates", doctor_treatment_plan.NewTreatmentTemplatesHandler(dataApi))

	// Doctor: Patient file APIs
	mux.Handle("/v1/doctor/patient/treatments", patient_file.NewDoctorPatientTreatmentsHandler(dataApi))
	mux.Handle("/v1/doctor/patient", patient_file.NewDoctorPatientHandler(dataApi, doseSpotService, smartyStreetsService))
	mux.Handle("/v1/doctor/patient/visits", patient_file.NewPatientVisitsHandler(dataApi))
	mux.Handle("/v1/doctor/patient/pharmacy", patient_file.NewDoctorUpdatePatientPharmacyHandler(dataApi))
	mux.Handle("/v1/doctor/treatment_plans", doctor_treatment_plan.NewDoctorTreatmentPlanHandler(dataApi, doseSpotService, erxStatusQueue, conf.ERxRouting))
	mux.Handle("/v1/doctor/treatment_plans/list", doctor_treatment_plan.NewListHandler(dataApi))
	mux.Handle("/v1/doctor/pharmacy", doctorPharmacySearchHandler)
	mux.Handle("/v1/doctor/visit/review", patient_file.NewDoctorPatientVisitReviewHandler(dataApi))
	mux.Handle("/v1/doctor/visit/diagnosis", patient_visit.NewDiagnosePatientHandler(dataApi, authAPI, conf.Environment))
	mux.Handle("/v1/doctor/visit/treatment/new", newTreatmentHandler)
	mux.Handle("/v1/doctor/visit/treatment/treatments", doctor_treatment_plan.NewTreatmentsHandler(dataApi, doseSpotService))
	mux.Handle("/v1/doctor/visit/treatment/medication_suggestions", autocompleteHandler)
	mux.Handle("/v1/doctor/visit/treatment/medication_strengths", medicationStrengthSearchHandler)
	mux.Handle("/v1/doctor/visit/treatment/medication_dispense_units", medicationDispenseUnitHandler)
	mux.Handle("/v1/doctor/visit/regimen", doctor_treatment_plan.NewRegimenHandler(dataApi))
	mux.Handle("/v1/doctor/visit/advice", doctor_treatment_plan.NewAdviceHandler(dataApi))
	mux.Handle("/v1/doctor/saved_messages", apiservice.NewDoctorSavedMessageHandler(dataApi))
	mux.Handle("/v1/doctor/patient/case/claim", doctor_queue.NewClaimPatientCaseAccessHandler(dataApi, metricsRegistry.Scope("doctor_queue")))

	// Miscellaneous APIs
	mux.Handle("/v1/content", staticContentHandler)
	mux.Handle("/v1/ping", pingHandler)
	mux.Handle("/v1/photo", photos.NewHandler(dataApi, stores["photos"]))
	mux.Handle("/v1/layouts/upload", layout.NewLayoutUploadHandler(dataApi))
	mux.Handle("/v1/app_event", app_event.NewHandler())

	var alog analytics.Logger
	if conf.Analytics.LogPath != "" {
		var err error
		alog, err = analytics.NewFileLogger(conf.Analytics.LogPath, conf.Analytics.MaxEvents, time.Duration(conf.Analytics.MaxAge)*time.Second)
		if err != nil {
			log.Fatal(err)
		}
		if err := alog.Start(); err != nil {
			log.Fatal(err)
		}
	} else {
		alog = analytics.NullLogger{}
	}
	analyticsHandler, err := analytics.NewHandler(alog, metricsRegistry.Scope("analytics.event.client"))
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/v1/event/client", analyticsHandler)

	// add the api to create demo visits to every environment except production
	if conf.Environment != "prod" {
		mux.Handle("/v1/doctor/demo/patient_visit", demo.NewHandler(dataApi, cloudStorageApi, conf.AWSRegion, conf.Environment))
		mux.Handle("/v1/doctor/demo/favorite_treatment_plan", demo.NewFavoriteTreatmentPlanHandler(dataApi))
	}

	if conf.ERxRouting {
		app_worker.StartWorkerToUpdatePrescriptionStatusForPatient(dataApi, doseSpotService, erxStatusQueue, metricsRegistry.Scope("check_erx_status"))
		app_worker.StartWorkerToCheckForRefillRequests(dataApi, doseSpotService, metricsRegistry.Scope("check_rx_refill_requests"), conf.Environment)
	}

	// This helps to ensure that we are only surfacing errors to client in the dev environment
	environment.SetCurrent(conf.Environment)

	// seeding random number generator based on time the main function runs
	rand.Seed(time.Now().UTC().UnixNano())

	return httputil.CompressResponse(httputil.DecompressRequest(mux))
}
