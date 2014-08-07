package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/libs/httputil"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_worker"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"

	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/payment/stripe"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/notify"

	restapi_router "github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/surescripts/pharmacy"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
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
	// Redirect any unknown subdomains to the website. This will most likely be a
	// bare domain without a subdomain (e.g. sprucehealth.com -> www.sprucehealth.com).
	router.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.Host, ".")
		if len(parts) < 2 {
			http.NotFound(w, r)
			return
		}
		host := strings.Join(parts[len(parts)-2:], ".")
		http.Redirect(w, r, fmt.Sprintf("https://%s.%s", conf.WebSubdomain, host), http.StatusMovedPermanently)
		return
	}))

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
		SecretKey:      conf.Stripe.SecretKey,
		PublishableKey: conf.Stripe.PublishableKey,
	}

	return router.New(&router.Config{
		DataAPI:           dataApi,
		AuthAPI:           authAPI,
		TwilioCli:         twilioCli,
		FromNumber:        conf.Twilio.FromNumber,
		EmailService:      email.NewService(conf.Email, metricsRegistry.Scope("email")),
		SupportEmail:      conf.Support.CustomerSupportEmail,
		WebSubdomain:      conf.WebSubdomain,
		StaticResourceURL: conf.StaticResourceURL,
		StripeCli:         stripeCli,
		Signer:            signer,
		Stores:            stores,
		WebPassword:       conf.WebPassword,
		MetricsRegistry:   metricsRegistry.Scope("www"),
	})
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

	doseSpotService := erx.NewDoseSpotService(conf.DoseSpot.ClinicId, conf.DoseSpot.UserId, conf.DoseSpot.ClinicKey, conf.DoseSpot.SOAPEndpoint, conf.DoseSpot.APIEndpoint, metricsRegistry.Scope("dosespot_api"))
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

	// This helps to ensure that we are only surfacing errors to client in the dev environment
	environment.SetCurrent(conf.Environment)

	mux := restapi_router.New(&restapi_router.Config{
		DataAPI:                  dataApi,
		AuthAPI:                  authAPI,
		AddressValidationAPI:     smartyStreetsService,
		ZipcodeToCityStateMapper: conf.ZipCodeToCityStateMapper,
		PharmacySearchAPI:        surescriptsPharmacySearch,
		SNSClient:                snsClient,
		PaymentAPI:               stripeService,
		NotifyConfigs:            conf.NotifiyConfigs,
		NotificationManager:      notificationManager,
		ERxStatusQueue:           erxStatusQueue,
		ERxAPI:                   doseSpotService,
		EmailService:             emailService,
		MetricsRegistry:          metricsRegistry,
		TwilioClient:             twilioCli,
		CloudStorageAPI:          cloudStorageApi,
		Stores:                   stores,
		ERxRouting:               conf.ERxRouting,
		JBCQMinutesThreshold:     conf.JBCQMinutesThreshold,
		CustomerSupportEmail:     conf.Support.CustomerSupportEmail,
		TechnicalSupportEmail:    conf.Support.TechnicalSupportEmail,
		APISubdomain:             conf.APISubdomain,
		WebSubdomain:             conf.WebSubdomain,
		StaticContentURL:         conf.StaticContentBaseUrl,
		ContentBucket:            conf.ContentBucket,
		AWSRegion:                conf.AWSRegion,
		AnalyticsLogger:          alog,
	})

	// Start worker to check for expired items in the global case queue
	doctor_queue.StartClaimedItemsExpirationChecker(dataApi, metricsRegistry.Scope("doctor_queue"))
	if conf.ERxRouting {
		app_worker.StartWorkerToUpdatePrescriptionStatusForPatient(dataApi, doseSpotService, erxStatusQueue, metricsRegistry.Scope("check_erx_status"))
		app_worker.StartWorkerToCheckForRefillRequests(dataApi, doseSpotService, metricsRegistry.Scope("check_rx_refill_requests"))
		app_worker.StartWorkerToCheckRxErrors(dataApi, doseSpotService, metricsRegistry.Scope("check_rx_errors"))
	}

	// seeding random number generator based on time the main function runs
	rand.Seed(time.Now().UTC().UnixNano())

	return httputil.CompressResponse(httputil.DecompressRequest(mux))
}
