package main

import (
	"database/sql"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	restapi_router "github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/demo"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/medrecord"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/schedmsg"
	"github.com/sprucehealth/backend/surescripts/pharmacy"
	"github.com/sprucehealth/backend/third_party/github.com/cookieo9/resources-go"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/third_party/github.com/subosito/twilio"
	"github.com/sprucehealth/backend/www"
	"github.com/sprucehealth/backend/www/router"
)

const (
	defaultMaxInMemoryPhotoMB = 2
)

func connectDB(conf *Config) *sql.DB {
	db, err := conf.DB.ConnectMySQL(conf.BaseConfig)
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

	environment.SetCurrent(conf.Environment)

	awsAuth, err := conf.AWSAuth()
	if err != nil {
		log.Fatalf("Failed to get AWS auth: %+v", err)
	}
	stores := storage.StoreMap{}
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

	dataApi, err := api.NewDataService(db, conf.APIDomain)
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

	authAPI, err := api.NewAuthAPI(
		db,
		time.Duration(conf.RegularAuth.ExpireDuration)*time.Second,
		time.Duration(conf.RegularAuth.RenewDuration)*time.Second,
		time.Duration(conf.ExtendedAuth.ExpireDuration)*time.Second,
		time.Duration(conf.ExtendedAuth.RenewDuration)*time.Second,
		api.NewBcryptHasher(0),
	)
	if err != nil {
		log.Fatal(err)
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

	restAPIMux := buildRESTAPI(&conf, dataApi, authAPI, signer, stores, metricsRegistry)
	webMux := buildWWW(&conf, dataApi, authAPI, signer, stores, metricsRegistry, conf.OnboardingURLExpires)

	// Remove port numbers since the muxer doesn't include them in the match
	apiDomain := conf.APIDomain
	if i := strings.IndexByte(apiDomain, ':'); i > 0 {
		apiDomain = apiDomain[:i]
	}
	webDomain := conf.WebDomain
	if i := strings.IndexByte(webDomain, ':'); i > 0 {
		webDomain = webDomain[:i]
	}

	router := mux.NewRouter()
	router.Host(apiDomain).Handler(restAPIMux)
	router.Host(webDomain).Handler(webMux)
	// Redirect any unknown domains to the website. This will most likely be a
	// bare domain without a subdomain (e.g. sprucehealth.com -> www.sprucehealth.com).
	router.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		http.Redirect(w, r, "https://"+conf.WebDomain, http.StatusMovedPermanently)
		return
	}))

	conf.SetupLogging()

	serve(&conf, router)
}

type twilioSMSAPI struct {
	*twilio.Client
}

func (sms *twilioSMSAPI) Send(fromNumber, toNumber, text string) error {
	_, _, err := sms.Client.Messages.SendSMS(fromNumber, toNumber, text)
	return err
}

func buildWWW(conf *Config, dataApi api.DataAPI, authAPI api.AuthAPI, signer *common.Signer, stores storage.StoreMap, metricsRegistry metrics.Registry, onboardingURLExpires int64) http.Handler {
	twilioCli, err := conf.Twilio.Client()
	if err != nil {
		if conf.Debug {
			log.Println(err.Error())
		} else {
			log.Fatal(err.Error())
		}
	}

	stripeCli := &stripe.StripeService{
		SecretKey:      conf.Stripe.SecretKey,
		PublishableKey: conf.Stripe.PublishableKey,
	}

	templateLoader := www.NewTemplateLoader(func(path string) (io.ReadCloser, error) {
		return resources.DefaultBundle.Open("templates/" + path)
	})

	var analyticsDB *sql.DB
	if conf.AnalyticsDB.Host != "" {
		analyticsDB, err = conf.AnalyticsDB.ConnectPostgres()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		golog.Warningf("No analytics database configured")
	}

	return router.New(&router.Config{
		DataAPI:              dataApi,
		AuthAPI:              authAPI,
		AnalyticsDB:          analyticsDB,
		TwilioCli:            twilioCli,
		FromNumber:           conf.Twilio.FromNumber,
		EmailService:         email.NewService(dataApi, conf.Email, metricsRegistry.Scope("email")),
		SupportEmail:         conf.Support.CustomerSupportEmail,
		WebDomain:            conf.WebDomain,
		StaticResourceURL:    conf.StaticResourceURL,
		StripeCli:            stripeCli,
		Signer:               signer,
		Stores:               stores,
		WebPassword:          conf.WebPassword,
		TemplateLoader:       templateLoader,
		OnboardingURLExpires: onboardingURLExpires,
		MetricsRegistry:      metricsRegistry.Scope("www"),
	})
}

func buildRESTAPI(conf *Config, dataApi api.DataAPI, authAPI api.AuthAPI, signer *common.Signer, stores storage.StoreMap, metricsRegistry metrics.Registry) http.Handler {
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

	emailService := email.NewService(dataApi, conf.Email, metricsRegistry.Scope("email"))
	surescriptsPharmacySearch, err := pharmacy.NewSurescriptsPharmacySearch(conf.PharmacyDB, conf.Environment)
	if err != nil {
		if conf.Debug {
			log.Printf("Unable to initialize pharmacy search: %s", err)
		} else {
			log.Fatalf("Unable to initialize pharmacy search: %s", err)
		}
	}

	var erxStatusQueue *common.SQSQueue
	if conf.ERxStatusQueue != "" {
		var err error
		erxStatusQueue, err = common.NewQueue(awsAuth, aws.Regions[conf.AWSRegion], conf.ERxStatusQueue)
		if err != nil {
			log.Fatalf("Unable to get erx queue for sending prescriptions to: %s", err.Error())
		}
	} else if conf.Debug {
		erxStatusQueue = &common.SQSQueue{
			QueueService: &sqs.Mock{},
			QueueUrl:     "ERxStatusQueue",
		}
	} else if conf.ERxRouting {
		log.Fatal("ERxStatusQueue not configured but ERxRouting is enabled")
	}

	var erxRoutingQueue *common.SQSQueue
	if conf.ERxRoutingQueue != "" {
		var err error
		erxRoutingQueue, err = common.NewQueue(awsAuth, aws.Regions[conf.AWSRegion], conf.ERxRoutingQueue)
		if err != nil {
			log.Fatalf("Unable to get erx queue for sending prescriptions to: %s", err.Error())
		}
	} else if conf.Debug {
		erxRoutingQueue = &common.SQSQueue{
			QueueService: &sqs.Mock{},
			QueueUrl:     "ERXRoutingQueue",
		}
	} else if conf.ERxRouting {
		log.Fatal("ERxRoutingQueue not configured but ERxRouting is enabled")
	}

	var medicalRecordQueue *common.SQSQueue
	if conf.MedicalRecordQueue != "" {
		medicalRecordQueue, err = common.NewQueue(awsAuth, aws.Regions[conf.AWSRegion], conf.MedicalRecordQueue)
		if err != nil {
			log.Fatalf("Failed to get queue for medical record requests: %s", err.Error())
		}
	} else if !conf.Debug {
		log.Fatal("MedicalRecordQueue not configured")
	} else {
		medicalRecordQueue = &common.SQSQueue{
			QueueService: &sqs.Mock{},
			QueueUrl:     "MedicalRecord",
		}
	}

	var visitQueue *common.SQSQueue
	if conf.VisitQueue != "" {
		visitQueue, err = common.NewQueue(awsAuth, aws.Regions[conf.AWSRegion], conf.VisitQueue)
		if err != nil {
			log.Fatalf("Failed to get queue for charging visits: %s", err.Error())
		}
	} else if !conf.Debug {
		log.Fatal("VisitQueue not configured")
	} else {
		visitQueue = &common.SQSQueue{
			QueueService: &sqs.Mock{},
			QueueUrl:     "Visit",
		}
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

	doseSpotService := erx.NewDoseSpotService(conf.DoseSpot.ClinicId, conf.DoseSpot.ProxyId, conf.DoseSpot.ClinicKey, conf.DoseSpot.SOAPEndpoint, conf.DoseSpot.APIEndpoint, metricsRegistry.Scope("dosespot_api"))
	notificationManager := notify.NewManager(dataApi, snsClient, twilioCli, emailService,
		conf.Twilio.FromNumber, conf.AlertEmail, conf.NotifiyConfigs, metricsRegistry.Scope("notify"))
	cloudStorageApi := api.NewCloudStorageService(awsAuth)

	stripeService := &stripe.StripeService{}
	if conf.TestStripe != nil && conf.TestStripe.SecretKey != "" {
		if conf.Environment == "prod" {
			golog.Warningf("Using test stripe key in production for patient")
		}
		stripeService.SecretKey = conf.TestStripe.SecretKey
	} else {
		stripeService.SecretKey = conf.Stripe.SecretKey
	}

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

	mux := restapi_router.New(&restapi_router.Config{
		DataAPI:                  dataApi,
		AuthAPI:                  authAPI,
		AuthTokenExpiration:      time.Duration(conf.RegularAuth.ExpireDuration) * time.Second,
		AddressValidationAPI:     smartyStreetsService,
		ZipcodeToCityStateMapper: conf.ZipCodeToCityStateMapper,
		PharmacySearchAPI:        surescriptsPharmacySearch,
		SNSClient:                snsClient,
		PaymentAPI:               stripeService,
		NotifyConfigs:            conf.NotifiyConfigs,
		MinimumAppVersionConfigs: conf.MinimumAppVersionConfigs,
		DosespotConfig:           conf.DoseSpot,
		NotificationManager:      notificationManager,
		ERxRoutingQueue:          erxRoutingQueue,
		ERxStatusQueue:           erxStatusQueue,
		ERxAPI:                   doseSpotService,
		VisitQueue:               visitQueue,
		MedicalRecordQueue:       medicalRecordQueue,
		EmailService:             emailService,
		MetricsRegistry:          metricsRegistry,
		SMSAPI:                   &twilioSMSAPI{twilioCli},
		CloudStorageAPI:          cloudStorageApi,
		Stores:                   stores,
		MaxCachedItems:           2000,
		ERxRouting:               conf.ERxRouting,
		JBCQMinutesThreshold:     conf.JBCQMinutesThreshold,
		CustomerSupportEmail:     conf.Support.CustomerSupportEmail,
		TechnicalSupportEmail:    conf.Support.TechnicalSupportEmail,
		APIDomain:                conf.APIDomain,
		WebDomain:                conf.WebDomain,
		StaticContentURL:         conf.StaticContentBaseUrl,
		StaticResourceURL:        conf.StaticResourceURL,
		ContentBucket:            conf.ContentBucket,
		AWSRegion:                conf.AWSRegion,
		AnalyticsLogger:          alog,
		TwoFactorExpiration:      conf.TwoFactorExpiration,
		SMSFromNumber:            conf.Twilio.FromNumber,
	})

	// Start worker to check for expired items in the global case queue
	doctor_queue.StartClaimedItemsExpirationChecker(dataApi, metricsRegistry.Scope("doctor_queue"))
	if conf.ERxRouting {
		app_worker.StartWorkerToUpdatePrescriptionStatusForPatient(dataApi, doseSpotService, erxStatusQueue, metricsRegistry.Scope("check_erx_status"))
		app_worker.StartWorkerToCheckForRefillRequests(dataApi, doseSpotService, metricsRegistry.Scope("check_rx_refill_requests"))
		app_worker.StartWorkerToCheckRxErrors(dataApi, doseSpotService, metricsRegistry.Scope("check_rx_errors"))
		doctor_treatment_plan.StartWorker(dataApi, doseSpotService, erxRoutingQueue, erxStatusQueue, 0, metricsRegistry.Scope("erx_route"))
	}

	medrecord.StartWorker(dataApi, medicalRecordQueue, emailService, conf.Support.CustomerSupportEmail, conf.APIDomain, conf.WebDomain, signer, stores.MustGet("medicalrecords"), stores.MustGet("media"), time.Duration(conf.RegularAuth.ExpireDuration)*time.Second)
	patient_visit.StartWorker(dataApi, stripeService, emailService, visitQueue, metricsRegistry.Scope("visit_queue"), conf.VisitWorkerTimePeriodSeconds, conf.Support.CustomerSupportEmail)
	schedmsg.StartWorker(dataApi, emailService, metricsRegistry.Scope("sched_msg"), 0)

	if !environment.IsProd() {
		demo.StartWorker(dataApi, conf.APIDomain, conf.AWSRegion, 0)
	}

	// seeding random number generator based on time the main function runs
	rand.Seed(time.Now().UTC().UnixNano())

	return httputil.CompressResponse(httputil.DecompressRequest(mux))
}
