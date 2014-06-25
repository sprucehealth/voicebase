package main

import (
	"carefront/address"
	"carefront/analytics"
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_worker"
	"carefront/common"
	"carefront/common/config"
	"carefront/demo"
	"carefront/doctor_queue"
	"carefront/doctor_treatment_plan"
	"carefront/email"
	"carefront/homelog"
	"carefront/layout"
	"carefront/libs/aws"
	"carefront/libs/aws/sns"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"carefront/libs/maps"
	"carefront/libs/payment/stripe"
	"carefront/libs/pharmacy"
	"carefront/messages"
	"carefront/notify"
	"carefront/passreset"
	"carefront/patient"
	"carefront/patient_file"
	"carefront/patient_visit"
	"carefront/photos"
	"carefront/reslib"
	"carefront/support"
	"carefront/treatment_plan"
	"carefront/www/router"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/samuel/go-metrics/metrics"
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
		golog.SetLevel(golog.DEBUG)
	} else if conf.Environment == "dev" {
		golog.SetLevel(golog.INFO)
	}

	conf.Validate()

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

	restAPIMux := buildRESTAPI(&conf, dataApi, authAPI, metricsRegistry)
	webMux := buildWWW(&conf, dataApi, authAPI, metricsRegistry)

	router := mux.NewRouter()
	router.Host(conf.APISubdomain + ".{domain:.+}").Handler(restAPIMux)
	router.Host(conf.WebSubdomain + ".{domain:.+}").Handler(webMux)

	conf.SetupLogging()

	serve(&conf, router)
}

func buildWWW(conf *Config, dataApi api.DataAPI, authAPI api.AuthAPI, metricsRegistry metrics.Registry) http.Handler {
	twilioCli, err := conf.Twilio.Client()
	if err != nil {
		if conf.Debug {
			log.Println(err.Error())
		} else {
			log.Fatal(err.Error())
		}
	}

	return router.New(dataApi, authAPI, twilioCli, conf.Twilio.FromNumber, email.NewService(conf.Email, metricsRegistry.Scope("email")), conf.Support.CustomerSupportEmail, conf.WebSubdomain, metricsRegistry.Scope("www"))
}

func buildRESTAPI(conf *Config, dataApi api.DataAPI, authAPI api.AuthAPI, metricsRegistry metrics.Registry) http.Handler {
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
	doseSpotService := erx.NewDoseSpotService(conf.DoseSpot.ClinicId, conf.DoseSpot.UserId, conf.DoseSpot.ClinicKey, metricsRegistry.Scope("dosespot_api"))

	notificationManager := notify.NewManager(dataApi, snsClient, twilioCli, emailService,
		conf.Twilio.FromNumber, conf.AlertEmail, conf.NotifiyConfigs, metricsRegistry.Scope("notify"))

	// Initialize listeneres
	homelog.InitListeners(dataApi, notificationManager)
	doctor_queue.InitListeners(dataApi, notificationManager)
	doctor_treatment_plan.InitListeners(dataApi)
	notify.InitListeners(dataApi)
	support.InitListeners(conf.Support.TechnicalSupportEmail, conf.Support.CustomerSupportEmail, notificationManager)

	// Start worker to check for expired items in the global case queue
	doctor_queue.StartClaimedItemsExpirationChecker(dataApi)

	cloudStorageApi := api.NewCloudStorageService(awsAuth)
	checkElligibilityHandler := &apiservice.CheckCareProvidingElligibilityHandler{DataApi: dataApi, AddressValidationApi: smartyStreetsService, StaticContentUrl: conf.StaticContentBaseUrl}
	updatePatientBillingAddress := &apiservice.UpdatePatientAddressHandler{DataApi: dataApi, AddressType: apiservice.BILLING_ADDRESS_TYPE}
	updatePatientPharmacyHandler := &apiservice.UpdatePatientPharmacyHandler{DataApi: dataApi, PharmacySearchService: pharmacy.GooglePlacesPharmacySearchService(0)}
	authenticateDoctorHandler := &apiservice.DoctorAuthenticationHandler{DataApi: dataApi, AuthApi: authAPI}
	signupDoctorHandler := &apiservice.SignupDoctorHandler{DataApi: dataApi, AuthApi: authAPI}
	autocompleteHandler := &apiservice.AutocompleteHandler{DataApi: dataApi, ERxApi: doseSpotService, Role: api.PATIENT_ROLE}
	doctorTreatmentSuggestionHandler := &apiservice.AutocompleteHandler{DataApi: dataApi, ERxApi: doseSpotService, Role: api.DOCTOR_ROLE}
	medicationStrengthSearchHandler := &apiservice.MedicationStrengthSearchHandler{DataApi: dataApi, ERxApi: doseSpotService}
	newTreatmentHandler := &apiservice.NewTreatmentHandler{DataApi: dataApi, ERxApi: doseSpotService}
	medicationDispenseUnitHandler := &apiservice.MedicationDispenseUnitsHandler{DataApi: dataApi}
	pharmacySearchHandler := &apiservice.PharmacyTextSearchHandler{PharmacySearchService: pharmacy.GooglePlacesPharmacySearchService(0), DataApi: dataApi, MapsService: mapsService}
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

	doctorUpdatePatientPharmacyHandler := &apiservice.DoctorUpdatePatientPharmacyHandler{
		DataApi: dataApi,
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
	mux.Handle("/v1/patient/pharmacy", updatePatientPharmacyHandler)
	mux.Handle("/v1/patient/isauthenticated", apiservice.NewIsAuthenticatedHandler(authAPI))
	mux.Handle("/v1/reset_password", passreset.NewForgotPasswordHandler(dataApi, authAPI, emailService, conf.Support.CustomerSupportEmail, conf.WebSubdomain))
	mux.Handle("/v1/credit_card", patientCardsHandler)
	mux.Handle("/v1/credit_card/default", patientCardsHandler)
	mux.Handle("/v1/authenticate", patient.NewAuthenticationHandler(dataApi, authAPI, pharmacy.GooglePlacesPharmacySearchService(0), conf.StaticContentBaseUrl))
	mux.Handle("/v1/logout", patient.NewAuthenticationHandler(dataApi, authAPI, pharmacy.GooglePlacesPharmacySearchService(0), conf.StaticContentBaseUrl))

	// Patient: Home APIs
	mux.Handle("/v1/patient/home", homelog.NewListHandler(dataApi))
	mux.Handle("/v1/patient/home/dismiss", homelog.NewDismissHandler(dataApi))

	// Patient: Patient Visit APIs
	mux.Handle("/v1/check_eligibility", checkElligibilityHandler)
	mux.Handle("/v1/patient/visit", patient_visit.NewPatientVisitHandler(dataApi, authAPI))
	mux.Handle("/v1/patient/visit/answer", patient_visit.NewAnswerIntakeHandler(dataApi))
	mux.Handle("/v1/patient/visit/photo_answer", patient_visit.NewPhotoAnswerIntakeHandler(dataApi))

	mux.Handle("/v1/treatment_plan", treatment_plan.NewTreatmentPlanHandler(dataApi))
	mux.Handle("/v1/treatment_guide", treatment_plan.NewTreatmentGuideHandler(dataApi))
	mux.Handle("/v1/autocomplete", autocompleteHandler)
	mux.Handle("/v1/pharmacy_search", pharmacySearchHandler)

	// Patient/Doctor: Resource guide APIs
	mux.Handle("/v1/resourceguide", reslib.NewHandler(dataApi))
	mux.Handle("/v1/resourceguide/list", reslib.NewListHandler(dataApi))

	// Patient/Doctor: Message APIs
	mux.Handle("/v1/case/messages", messages.NewHandler(dataApi))
	mux.Handle("/v1/case/messages/list", messages.NewListHandler(dataApi))
	mux.Handle("/v1/case/messages/read", messages.NewReadHandler(dataApi))

	// Doctor: Account APIs
	mux.Handle("/v1/doctor/signup", signupDoctorHandler)
	mux.Handle("/v1/doctor/authenticate", authenticateDoctorHandler)
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
	mux.Handle("/v1/doctor/visit/treatment/medication_suggestions", doctorTreatmentSuggestionHandler)
	mux.Handle("/v1/doctor/visit/treatment/medication_strengths", medicationStrengthSearchHandler)
	mux.Handle("/v1/doctor/visit/treatment/medication_dispense_units", medicationDispenseUnitHandler)
	mux.Handle("/v1/doctor/visit/regimen", doctor_treatment_plan.NewRegimenHandler(dataApi))
	mux.Handle("/v1/doctor/visit/advice", doctor_treatment_plan.NewAdviceHandler(dataApi))
	mux.Handle("/v1/doctor/saved_messages", apiservice.NewDoctorSavedMessageHandler(dataApi))
	mux.Handle("v1/doctor/patient/case/claim", doctor_queue.NewClaimPatientCaseAccessHandler(dataApi))

	// Miscellaneous APIs
	mux.Handle("/v1/content", staticContentHandler)
	mux.Handle("/v1/ping", pingHandler)
	mux.Handle("/v1/photo", photos.NewHandler(dataApi, awsAuth, conf.PhotoBucket, conf.AWSRegion))
	mux.Handle("/v1/layouts/upload", layout.NewLayoutUploadHandler(dataApi))

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
	apiservice.IsDev = (conf.Environment == "dev")

	return mux
}
