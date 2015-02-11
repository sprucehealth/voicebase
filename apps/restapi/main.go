package main

import (
	"database/sql"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/armon/consul-api"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/cookieo9/resources-go"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-librato/librato"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/subosito/twilio"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/gopkgs.com/memcache.v2"
	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/analytics/analisteners"
	"github.com/sprucehealth/backend/api"
	restapi_router "github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/consul"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/demo"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/elasticache"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/medrecord"
	"github.com/sprucehealth/backend/misc"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/schedmsg"
	"github.com/sprucehealth/backend/surescripts/pharmacy"
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

	dispatcher := dispatch.New()

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

	dataAPI, err := api.NewDataService(db)
	if err != nil {
		log.Fatalf("Unable to initialize data service layer: %s", err)
	}

	metricsRegistry := metrics.NewRegistry()

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

	var smsAPI api.SMSAPI
	if twilioCli, err := conf.Twilio.Client(); err == nil {
		smsAPI = &twilioSMSAPI{twilioCli}
	} else if conf.Debug {
		log.Println(err.Error())
		smsAPI = loggingSMSAPI{}
	} else {
		log.Fatal(err.Error())
	}

	conf.StartReporters(metricsRegistry)

	var consulService *consul.Service
	if conf.Consul.ConsulAddress != "" {
		consulClient, err := consulapi.NewClient(&consulapi.Config{
			Address:    conf.Consul.ConsulAddress,
			HttpClient: http.DefaultClient,
		})
		if err != nil {
			golog.Fatalf("Unable to instantiate new consul client: %s", err)
		}

		consulService, err = consul.RegisterService(consulClient, conf.Consul.ConsulServiceID, "restapi", nil, 0)
		if err != nil {
			log.Fatalf("Failed to register service with Consul: %s", err.Error())
		}
	} else {
		golog.Warningf("Consul address not specified")
	}

	defer func() {
		if consulService != nil {
			consulService.Deregister()
		}
	}()

	sigKeys := make([][]byte, len(conf.SecretSignatureKeys))
	for i, k := range conf.SecretSignatureKeys {
		// No reason to decode the keys to binary. They'll be slightly longer
		// as ascii but include no less entropy.
		sigKeys[i] = []byte(k)
	}
	signer := &common.Signer{
		Keys: sigKeys,
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
		if conf.Debug {
			alog = analytics.DebugLogger{}
		} else {
			alog = analytics.NullLogger{}
		}
	}
	analisteners.InitListeners(alog, dispatcher)

	if conf.OfficeNotifySNSTopic != "" {
		awsAuth, err := conf.AWSAuth()
		if err != nil {
			log.Fatalf("Failed to get AWS auth: %+v", err)
		}
		snsClient := &sns.SNS{
			Region: aws.USEast,
			Client: &aws.Client{
				Auth: awsAuth,
			},
		}
		InitNotifyListener(dispatcher, snsClient, conf.OfficeNotifySNSTopic)
	}

	var memcacheCli *memcache.Client
	if conf.Memcached != nil {
		if m := conf.Memcached["cache"]; m != nil {
			var servers memcache.Servers
			if m.DiscoveryHost != "" {
				if m.DiscoveryInterval <= 0 {
					m.DiscoveryInterval = 60 * 5
				}
				d, err := elasticache.NewDiscoverer(m.DiscoveryHost, time.Second*time.Duration(m.DiscoveryInterval))
				if err != nil {
					log.Fatalf("Failed to discover memcached hosts: %s", err.Error())
				}
				servers = NewElastiCacheServers(d)
			} else {
				servers = NewHRWServer(m.Hosts)
			}
			memcacheCli = memcache.NewFromServers(servers)
		}
	}

	rateLimiters := ratelimit.KeyedRateLimiters(make(map[string]ratelimit.KeyedRateLimiter))
	if memcacheCli != nil {
		for n, c := range conf.RateLimiters {
			rateLimiters[n] = ratelimit.NewMemcache(memcacheCli, c.Max, c.Period)
		}
	}

	doseSpotService := erx.NewDoseSpotService(conf.DoseSpot.ClinicID, conf.DoseSpot.ProxyID, conf.DoseSpot.ClinicKey, conf.DoseSpot.SOAPEndpoint, conf.DoseSpot.APIEndpoint, metricsRegistry.Scope("dosespot_api"))

	diagnosisAPI, err := diagnosis.NewService(conf.DiagnosisDB)
	if err != nil {
		if conf.Debug {
			golog.Warningf("Failed to setup diagnosis service: %s", err.Error())
		} else {
			golog.Fatalf("Failed to setup diagnosis service: %s", err.Error())
		}
	}

	restAPIMux := buildRESTAPI(&conf, dataAPI, authAPI, diagnosisAPI, smsAPI, doseSpotService, dispatcher, consulService, signer, stores, rateLimiters, alog, metricsRegistry)
	webMux := buildWWW(&conf, dataAPI, authAPI, diagnosisAPI, smsAPI, doseSpotService, dispatcher, signer, stores, rateLimiters, alog, metricsRegistry, conf.OnboardingURLExpires)

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
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != apiDomain && r.Host != webDomain {
			// If apex domain (e.g. sprucehealth.com) then just rewrite host
			if idx := strings.IndexByte(r.Host, '.'); idx == strings.LastIndex(r.Host, ".") {
				u := *r.URL
				u.Scheme = "https"
				u.Host = webDomain
				http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
			} else {
				http.Redirect(w, r, "https://"+conf.WebDomain, http.StatusMovedPermanently)
			}
		} else {
			http.NotFound(w, r)
		}
		return
	})

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

type loggingSMSAPI struct{}

func (loggingSMSAPI) Send(fromNumber, toNumber, text string) error {
	golog.Infof("SMS: from=%s to=%s text=%s", fromNumber, toNumber, text)
	return nil
}

func buildWWW(conf *Config, dataAPI api.DataAPI, authAPI api.AuthAPI, diagnosisAPI diagnosis.API, smsAPI api.SMSAPI, eRxAPI erx.ERxAPI,
	dispatcher *dispatch.Dispatcher, signer *common.Signer, stores storage.StoreMap, rateLimiters ratelimit.KeyedRateLimiters,
	alog analytics.Logger, metricsRegistry metrics.Registry, onboardingURLExpires int64,
) http.Handler {
	stripeCli := &stripe.StripeService{
		SecretKey:      conf.Stripe.SecretKey,
		PublishableKey: conf.Stripe.PublishableKey,
	}

	templateLoader := www.NewTemplateLoader(func(path string) (io.ReadCloser, error) {
		return resources.DefaultBundle.Open("templates/" + path)
	})

	var err error
	var analyticsDB *sql.DB
	if conf.AnalyticsDB.Host != "" {
		analyticsDB, err = conf.AnalyticsDB.ConnectPostgres()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		golog.Warningf("No analytics database configured")
	}

	var lc *librato.Client
	if conf.Stats.LibratoToken != "" && conf.Stats.LibratoUsername != "" {
		lc = &librato.Client{
			Username: conf.Stats.LibratoUsername,
			Token:    conf.Stats.LibratoToken,
		}
	}

	return router.New(&router.Config{
		DataAPI:              dataAPI,
		AuthAPI:              authAPI,
		DiagnosisAPI:         diagnosisAPI,
		SMSAPI:               smsAPI,
		ERxAPI:               eRxAPI,
		Dispatcher:           dispatcher,
		AnalyticsDB:          analyticsDB,
		AnalyticsLogger:      alog,
		FromNumber:           conf.Twilio.FromNumber,
		EmailService:         email.NewService(dataAPI, conf.Email, metricsRegistry.Scope("email")),
		SupportEmail:         conf.Support.CustomerSupportEmail,
		WebDomain:            conf.WebDomain,
		StaticResourceURL:    conf.StaticResourceURL,
		StripeClient:         stripeCli,
		Signer:               signer,
		Stores:               stores,
		RateLimiters:         rateLimiters,
		WebPassword:          conf.WebPassword,
		TemplateLoader:       templateLoader,
		OnboardingURLExpires: onboardingURLExpires,
		TwoFactorExpiration:  conf.TwoFactorExpiration,
		ExperimentIDs:        conf.ExperimentID,
		LibratoClient:        lc,
		MetricsRegistry:      metricsRegistry.Scope("www"),
	})
}

type localLock struct {
	mu         sync.Mutex
	internalmu sync.Mutex
	isLocked   bool
}

func newLocalLock() api.LockAPI {
	return &localLock{}
}

func (l *localLock) Wait() bool {
	l.internalmu.Lock()
	defer l.internalmu.Unlock()
	l.mu.Lock()
	l.isLocked = true
	return true
}

func (l *localLock) Release() {
	l.internalmu.Lock()
	defer l.internalmu.Unlock()
	l.mu.Unlock()
	l.isLocked = false
}

func (l *localLock) Locked() bool {
	l.internalmu.Lock()
	defer l.internalmu.Unlock()
	return l.isLocked
}

func buildRESTAPI(conf *Config, dataAPI api.DataAPI, authAPI api.AuthAPI, diagnosisAPI diagnosis.API, smsAPI api.SMSAPI, eRxAPI erx.ERxAPI,
	dispatcher *dispatch.Dispatcher, consulService *consul.Service, signer *common.Signer, stores storage.StoreMap,
	rateLimiters ratelimit.KeyedRateLimiters, alog analytics.Logger, metricsRegistry metrics.Registry,
) http.Handler {
	awsAuth, err := conf.AWSAuth()
	if err != nil {
		log.Fatalf("Failed to get AWS auth: %+v", err)
	}

	emailService := email.NewService(dataAPI, conf.Email, metricsRegistry.Scope("email"))
	surescriptsPharmacySearch, err := pharmacy.NewSurescriptsPharmacySearch(conf.PharmacyDB)
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
			QueueURL:     "ERxStatusQueue",
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
			QueueURL:     "ERXRoutingQueue",
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
			QueueURL:     "MedicalRecord",
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
			QueueURL:     "Visit",
		}
	}

	snsClient := &sns.SNS{
		Region: aws.USEast,
		Client: &aws.Client{
			Auth: awsAuth,
		},
	}
	smartyStreetsService := &address.SmartyStreetsService{
		AuthID:    conf.SmartyStreets.AuthID,
		AuthToken: conf.SmartyStreets.AuthToken,
	}

	notificationManager := notify.NewManager(dataAPI, authAPI, snsClient, smsAPI, emailService,
		conf.Twilio.FromNumber, conf.NotifiyConfigs, metricsRegistry.Scope("notify"))

	stripeService := &stripe.StripeService{}
	if conf.TestStripe != nil && conf.TestStripe.SecretKey != "" {
		if conf.Environment == "prod" {
			golog.Warningf("Using test stripe key in production for patient")
		}
		stripeService.SecretKey = conf.TestStripe.SecretKey
	} else {
		stripeService.SecretKey = conf.Stripe.SecretKey
	}

	mux := restapi_router.New(&restapi_router.Config{
		DataAPI:                  dataAPI,
		AuthAPI:                  authAPI,
		Dispatcher:               dispatcher,
		AuthTokenExpiration:      time.Duration(conf.RegularAuth.ExpireDuration) * time.Second,
		AddressValidationAPI:     smartyStreetsService,
		PharmacySearchAPI:        surescriptsPharmacySearch,
		DiagnosisAPI:             diagnosisAPI,
		SNSClient:                snsClient,
		PaymentAPI:               stripeService,
		NotifyConfigs:            conf.NotifiyConfigs,
		MinimumAppVersionConfigs: conf.MinimumAppVersionConfigs,
		DosespotConfig:           conf.DoseSpot,
		NotificationManager:      notificationManager,
		ERxRoutingQueue:          erxRoutingQueue,
		ERxStatusQueue:           erxStatusQueue,
		ERxAPI:                   eRxAPI,
		VisitQueue:               visitQueue,
		MedicalRecordQueue:       medicalRecordQueue,
		EmailService:             emailService,
		MetricsRegistry:          metricsRegistry,
		SMSAPI:                   smsAPI,
		Stores:                   stores,
		RateLimiters:             rateLimiters,
		MaxCachedItems:           2000,
		ERxRouting:               conf.ERxRouting,
		NumDoctorSelection:       conf.NumDoctorSelection,
		JBCQMinutesThreshold:     conf.JBCQMinutesThreshold,
		CustomerSupportEmail:     conf.Support.CustomerSupportEmail,
		TechnicalSupportEmail:    conf.Support.TechnicalSupportEmail,
		APIDomain:                conf.APIDomain,
		WebDomain:                conf.WebDomain,
		StaticContentURL:         conf.StaticContentBaseURL,
		StaticResourceURL:        conf.StaticResourceURL,
		AWSRegion:                conf.AWSRegion,
		AnalyticsLogger:          alog,
		TwoFactorExpiration:      conf.TwoFactorExpiration,
		SMSFromNumber:            conf.Twilio.FromNumber,
	})

	if !environment.IsProd() {
		demo.NewWorker(
			dataAPI,
			newLock("service/restapi/training_cases", consulService, conf.Debug),
			conf.APIDomain,
			conf.AWSRegion,
		).Start()
	}

	notifyDoctorLock := newLock("service/restapi/notify_doctor", consulService, conf.Debug)
	refillRequestCheckLock := newLock("service/restapi/check_refill_request", consulService, conf.Debug)
	checkRxErrorsLock := newLock("service/restapi/check_rx_error", consulService, conf.Debug)

	// Start worker to check for expired items in the global case queue
	doctor_queue.StartClaimedItemsExpirationChecker(dataAPI, alog, metricsRegistry.Scope("doctor_queue"))
	if conf.ERxRouting {
		app_worker.NewERxStatusWorker(
			dataAPI,
			eRxAPI,
			dispatcher,
			erxStatusQueue,
			metricsRegistry.Scope("check_erx_status"),
		).Start()
		app_worker.NewRefillRequestWorker(
			dataAPI,
			eRxAPI,
			refillRequestCheckLock,
			dispatcher,
			metricsRegistry.Scope("check_rx_refill_requests"),
		).Start()
		app_worker.NewERxErrorWorker(
			dataAPI,
			eRxAPI,
			checkRxErrorsLock,
			metricsRegistry.Scope("check_rx_errors"),
		).Start()
		doctor_treatment_plan.StartWorker(dataAPI, eRxAPI, dispatcher, erxRoutingQueue, erxStatusQueue, 0, metricsRegistry.Scope("erx_route"))
	}

	medrecord.NewWorker(
		dataAPI,
		medicalRecordQueue,
		emailService,
		conf.Support.CustomerSupportEmail,
		conf.APIDomain,
		conf.WebDomain,
		signer,
		stores.MustGet("medicalrecords"),
		stores.MustGet("media"),
		time.Duration(conf.RegularAuth.ExpireDuration)*time.Second,
	).Start()

	schedmsg.StartWorker(dataAPI, authAPI, dispatcher, emailService, metricsRegistry.Scope("sched_msg"), 0)
	misc.StartWorker(dataAPI, metricsRegistry)

	cost.NewWorker(
		dataAPI,
		alog,
		dispatcher,
		stripeService,
		emailService,
		visitQueue,
		metricsRegistry.Scope("visit_queue"),
		conf.VisitWorkerTimePeriodSeconds,
		conf.Support.CustomerSupportEmail,
	).Start()

	doctor_queue.NewWorker(
		dataAPI,
		authAPI,
		notifyDoctorLock,
		notificationManager,
		metricsRegistry.Scope("notify_doctors"),
	).Start()

	// seeding random number generator based on time the main function runs
	rand.Seed(time.Now().UTC().UnixNano())

	return httputil.CompressResponse(httputil.DecompressRequest(mux))
}

func newLock(name string, consulService *consul.Service, isDebug bool) api.LockAPI {
	var lock api.LockAPI
	if consulService != nil {
		lock = consulService.NewLock(name, nil, time.Second*30)
	} else if isDebug || environment.IsDemo() || environment.IsDev() {
		lock = newLocalLock()
	} else {
		golog.Fatalf("Unable to setup lock due to lack of consul service")
	}

	return lock
}
