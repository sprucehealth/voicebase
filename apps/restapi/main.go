package main

import (
	"database/sql"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/armon/consul-api"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/gopkgs.com/memcache.v2"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/analytics/analisteners"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/consul"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/elasticache"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
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
			s := storage.NewS3(awsAuth, c.Region, c.Bucket, c.Prefix)
			s.LatchedExpire(c.LatchedExpire)
			stores[name] = s
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
	signer, err := sig.NewSigner(sigKeys, nil)
	if err != nil {
		log.Fatalf("Failed to create signer: %s", err.Error())
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

	restAPIMux := buildRESTAPI(
		&conf, dataAPI, authAPI, diagnosisAPI, smsAPI, doseSpotService, memcacheCli,
		dispatcher, consulService, signer, stores, rateLimiters, alog, metricsRegistry)
	webMux := buildWWW(&conf, dataAPI, authAPI, diagnosisAPI, smsAPI, doseSpotService,
		dispatcher, signer, stores, rateLimiters, alog, metricsRegistry,
		conf.OnboardingURLExpires)

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
