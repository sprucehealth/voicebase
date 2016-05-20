package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/rainycape/memcache"
	"github.com/rs/cors"
	"github.com/samuel/go-metrics/metrics"
	"github.com/samuel/go-metrics/reporter"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/analytics/analisteners"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/products"
	"github.com/sprucehealth/backend/cmd/svc/regimens"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/handlers"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/mediaproxy"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/rxguide"
	"github.com/sprucehealth/backend/events"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/factual"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mcutil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/storage"
	iproducts "github.com/sprucehealth/backend/svc/products"
	"github.com/sprucehealth/go-proxy-protocol/proxyproto"
	"golang.org/x/net/context"
)

const (
	applicationName = "regimens"
)

type mediaConfig struct {
	storageType      string
	s3Bucket         string
	s3Prefix         string
	localStoragePath string
	maxWidth         int
	maxHeight        int
}

var config struct {
	httpAddr      string
	proxyProtocol bool
	webDomain     string
	apiDomain     string
	env           string

	// Factual config
	factualKey    string
	factualSecret string

	// Media
	media mediaConfig

	// Media proxy
	mediaProxy              mediaConfig
	mediaProxyCacheDuration time.Duration

	// Memcached config
	mcDiscoveryHost     string
	mcDiscoveryInterval time.Duration
	mcHosts             string

	// AWS config
	awsDynamoDBEndpoint   string
	awsDynamoDBRegion     string
	awsDynamoDBDisableSSL bool
	awsAccessKey          string
	awsSecretKey          string
	awsToken              string

	// Amazon.com
	amzAccessKey    string
	amzSecretKey    string
	amzAssociateTag string

	// Regimens auth secret
	authSecret string

	// Metrics
	metricsSource   string
	libratoUsername string
	libratoToken    string

	// Analytics
	analyticsLogPath                  string
	analyticsMaxEvents                int
	analyticsFirehoseStreams          string
	analyticsFirehoseMaxBatchSize     int
	analyticsFirehoseMaxBatchDuration time.Duration
	analyticsDebug                    bool

	// CORS
	corsAllowAll bool
}

func init() {
	// Regimens service
	flag.StringVar(&config.httpAddr, "http", "0.0.0.0:8000", "listen for http on `host:port`")
	flag.BoolVar(&config.proxyProtocol, "proxyproto", false, "enabled proxy protocol")
	flag.StringVar(&config.authSecret, "auth_secret", "", "Secret to use in auth token generation")
	flag.StringVar(&config.webDomain, "web_domain", "", "The web domain used for link generation")
	flag.StringVar(&config.apiDomain, "api_domain", "", "The api domain used for link generation")
	flag.StringVar(&config.env, "env", "undefined", "`Environment`")

	// Factual
	flag.StringVar(&config.factualKey, "factual_key", "", "Factual API `key`")
	flag.StringVar(&config.factualSecret, "factual_secret", "", "Factual API `secret`")

	// Media
	flag.StringVar(&config.media.storageType, "media_storage_type", "local", "Storage type for regimen media")
	flag.StringVar(&config.media.s3Bucket, "media_s3_bucket", "", "S3 Bucket for media storage")
	flag.StringVar(&config.media.s3Prefix, "media_s3_prefix", "", "S3 path prefix for media storage")
	flag.StringVar(&config.media.localStoragePath, "media_local_path", "/tmp", "Local path to use when using local media storage")
	flag.IntVar(&config.media.maxWidth, "media_max_width", 0, "Maximum `width` of stored media (0 for unbounded)")
	flag.IntVar(&config.media.maxWidth, "media_max_height", 0, "Maximum `height` of stored media (0 for unbounded)")

	// Media proxy
	flag.StringVar(&config.mediaProxy.storageType, "mediaproxy_storage_type", "local", "Storage type for media proxy")
	flag.StringVar(&config.mediaProxy.s3Bucket, "mediaproxy_s3_bucket", "", "S3 Bucket for media proxy storage")
	flag.StringVar(&config.mediaProxy.s3Prefix, "mediaproxy_s3_prefix", "", "S3 path prefix for media proxy storage")
	flag.StringVar(&config.mediaProxy.localStoragePath, "mediaproxy_local_path", "/tmp", "Local path to use when using local media proxy storage")
	flag.IntVar(&config.mediaProxy.maxWidth, "mediaproxy_max_width", 0, "Maximum `width` of stored proxied media (0 for unbounded)")
	flag.IntVar(&config.mediaProxy.maxWidth, "mediaproxy_max_height", 0, "Maximum `height` of stored proxied media (0 for unbounded)")
	flag.DurationVar(&config.mediaProxyCacheDuration, "mediaproxy_cache_duration", time.Second*60*60*24*7, "Cache `expiration` for media proxy metadata")

	// Memcached
	flag.StringVar(&config.mcDiscoveryHost, "mc_discovery_host", "", "ElastiCache discovery `host`")
	flag.DurationVar(&config.mcDiscoveryInterval, "mc_discovery_internal", time.Minute*5, "ElastiCache discovery `interval`")
	flag.StringVar(&config.mcHosts, "mc_hosts", "", "Comma separated list of memcached `hosts` when not using ElastiCache discovery")

	// AWS
	flag.StringVar(&config.awsDynamoDBEndpoint, "aws_dynamodb_endpoint", "", "AWS Dynamo DB API `endpoint`")
	flag.StringVar(&config.awsDynamoDBRegion, "aws_dynamodb_region", "", "AWS Dynamo DB API `region`")
	flag.BoolVar(&config.awsDynamoDBDisableSSL, "aws_dynamodb_disable_ssl", false, "Disable SSL in the AWS DynamoDB client")
	flag.StringVar(&config.awsAccessKey, "aws_access_key", "", "AWS Credentials Access Key")
	flag.StringVar(&config.awsSecretKey, "aws_secret_key", "", "AWS Credentials Secret Key")
	flag.StringVar(&config.awsToken, "aws_token", "", "AWS Credentials Token")

	// Amazon.com
	flag.StringVar(&config.amzAccessKey, "amz_access_key", "", "Access `key` for Amazon affiliate products API")
	flag.StringVar(&config.amzSecretKey, "amz_secret_key", "", "Secret `key` for Amazon affiliate products API")
	flag.StringVar(&config.amzAssociateTag, "amz_associate_tag", "", "Amazon affiliate associate tag")

	// Metrics
	flag.StringVar(&config.metricsSource, "metrics_source", "", "`Source` for metrics (e.g. hostname)")
	flag.StringVar(&config.libratoUsername, "librato_username", "", "Librato metrics `username`")
	flag.StringVar(&config.libratoToken, "librato_token", "", "Librato metrics auth `token`")

	// Analytics
	flag.StringVar(&config.analyticsLogPath, "analytics_log.path", "", "the place to write the analytics log file")
	flag.BoolVar(&config.analyticsDebug, "analytics_debug", false, "enable debug functionality in analytics emission")
	flag.IntVar(&config.analyticsMaxEvents, "analytics_max_events", analytics.DefaultMaxFileEvents, "the max events per analytics log file")
	flag.StringVar(&config.analyticsFirehoseStreams, "analytics_firehose_streams", "", "Kinesis Firehose streams in the format 'category:stream,category:stream,...'")
	flag.IntVar(&config.analyticsFirehoseMaxBatchSize, "analytics_firehose_batch_maxsize", 8, "Kinesis Firehose max batch size before flushing")
	flag.DurationVar(&config.analyticsFirehoseMaxBatchDuration, "analytics_firehose_batch_maxduration", time.Second*5, "Kinesis Firehose max duration to batch events before flushing")

	// CORS
	flag.BoolVar(&config.corsAllowAll, "cors_allow_all", true, "Enable the * patterns on CORS")
}

func main() {
	log.SetFlags(log.Lshortfile)
	boot.ParseFlags("REGIMENS_")

	config.apiDomain = strings.TrimRight(config.apiDomain, "/")
	config.webDomain = strings.TrimRight(config.webDomain, "/")

	metricsRegistry := metrics.NewRegistry()
	_, handler := setupRouter(metricsRegistry)

	if config.metricsSource == "" {
		hostname, err := os.Hostname()
		if err == nil {
			config.metricsSource = fmt.Sprintf("%s-%s-%s", config.env, "regimensapi", hostname)
		} else {
			config.metricsSource = "regimensapi"
			golog.Warningf("Unable to get local hostname: %s", err)
		}
	}
	metricsRegistry.Add("runtime", metrics.RuntimeMetrics)
	if config.libratoUsername != "" && config.libratoToken != "" {
		statsReporter := reporter.NewLibratoReporter(
			metricsRegistry, time.Minute, true, config.libratoUsername,
			config.libratoToken, config.metricsSource)
		statsReporter.Start()
		defer statsReporter.Stop()
	}

	serve(handler)
}

func setupRouter(metricsRegistry metrics.Registry) (*mux.Router, httputil.ContextHandler) {
	var memcacheCli *memcache.Client
	if config.mcDiscoveryHost != "" {
		d, err := awsutil.NewElastiCacheDiscoverer(config.mcDiscoveryHost, config.mcDiscoveryInterval)
		if err != nil {
			golog.Fatalf("Failed to discover memcached hosts: %s", err.Error())
		}
		memcacheCli = memcache.NewFromServers(mcutil.NewElastiCacheServers(d))
	} else if config.mcHosts != "" {
		var hosts []string
		for _, h := range strings.Split(config.mcHosts, ",") {
			hosts = append(hosts, strings.TrimSpace(h))
		}
		memcacheCli = memcache.NewFromServers(mcutil.NewHRWServer(hosts))
	}

	var amz products.AmazonProductClient
	if config.amzAccessKey != "" && config.amzSecretKey != "" && config.amzAssociateTag != "" {
		var err error
		amz, err = products.NewAmazonProductsClient(config.amzAccessKey, config.amzSecretKey, config.amzAssociateTag)
		if err != nil {
			golog.Fatalf(err.Error())
		}
	} else {
		golog.Warningf("Amazon associate keys and/tag not set")
	}

	// Bootstrap analytics mechanisms
	dispatcher := dispatch.New()
	analisteners.InitListeners(applicationName, getAnalyticsLogger(metricsRegistry.Scope("analytics")), dispatcher, events.NullClient{})

	var productCache products.MemcacheClient
	if memcacheCli != nil {
		productCache = memcacheCli
	}
	dynamoDBClient := dynamodb.New(func() *session.Session {
		if config.awsDynamoDBEndpoint != "" {
			golog.Infof("AWS Dynamo DB Endpoint configured as %s...", config.awsDynamoDBEndpoint)
			dynamoConfig := &aws.Config{
				Region:     &config.awsDynamoDBRegion,
				DisableSSL: &config.awsDynamoDBDisableSSL,
				Endpoint:   &config.awsDynamoDBEndpoint,
			}
			if config.awsDynamoDBDisableSSL {
				dynamoConfig.DisableSSL = &config.awsDynamoDBDisableSSL
			}
			return session.New(dynamoConfig)
		}
		return awsSession()
	}())
	rxGuideSvc, err := rxguide.New(dynamoDBClient, config.env)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	regimenSvc, err := regimens.New(dynamoDBClient, dispatcher, config.env, config.authSecret)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	productsSvc := products.New(
		factual.New(config.factualKey, config.factualSecret),
		amz,
		productCache,
		metricsRegistry.Scope("productssvc"),
		map[string]iproducts.DAL{"rxguide": rxguide.AsProductDAL(rxGuideSvc, config.apiDomain, config.webDomain)})

	requestLogger := func(ctx context.Context, ev *httputil.RequestEvent) {
		av := &analytics.WebRequestEvent{
			Service:      applicationName,
			RequestID:    httputil.RequestID(ctx),
			Path:         ev.URL.Path,
			Timestamp:    analytics.Time(ev.Timestamp),
			StatusCode:   ev.StatusCode,
			Method:       ev.Request.Method,
			URL:          ev.URL.String(),
			RemoteAddr:   ev.RemoteAddr,
			ContentType:  ev.ResponseHeaders.Get("Content-Type"),
			UserAgent:    ev.Request.UserAgent(),
			Referrer:     ev.Request.Referer(),
			ResponseTime: int(ev.ResponseTime.Nanoseconds() / 1e3),
			Server:       ev.ServerHostname,
		}
		log := golog.Context(
			"Method", av.Method,
			"URL", av.URL,
			"UserAgent", av.UserAgent,
			"RequestID", av.RequestID,
			"RemoteAddr", av.RemoteAddr,
			"StatusCode", av.StatusCode,
		)
		if ev.Panic != nil {
			log.Criticalf("http: panic: %v\n%s", ev.Panic, ev.StackTrace)
		} else {
			log.Infof("regimens-apirequest")
		}
		dispatcher.PublishAsync(av)
	}

	router := mux.NewRouter().StrictSlash(true)

	mediaStore := getMediaStore(config.media, "")
	mediaStoreCache := getMediaStore(config.media, "cache")
	mediaSvc := media.New(mediaStore, mediaStoreCache, config.media.maxWidth, config.media.maxHeight)
	mediaHandler := handlers.NewMedia(config.apiDomain, mediaSvc, metricsRegistry.Scope("media"))

	proxyMediaStore := getMediaStore(config.mediaProxy, "")
	proxyMediaStoreCache := getMediaStore(config.mediaProxy, "cache")
	proxyMediaSvc := media.New(proxyMediaStore, proxyMediaStoreCache, config.mediaProxy.maxWidth, config.mediaProxy.maxHeight)
	var proxyDAL mediaproxy.DAL
	proxyDAL = mediaproxy.NewMemoryDAL()
	if dynamoDBClient != nil {
		proxyDAL, err = mediaproxy.NewDynamoDBDAL(dynamoDBClient, config.env, metricsRegistry.Scope("mediaproxy/dynamodb"))
		if err != nil {
			golog.Fatalf("Failed to init mediaproxy DynamoDB DAL: %s", err)
		}
	}
	if memcacheCli != nil {
		proxyDAL = mediaproxy.NewCacheDAL(memcacheCli, proxyDAL, config.mediaProxyCacheDuration, metricsRegistry.Scope("mediaproxy/cache"))
	}
	proxySvc := mediaproxy.New(proxyMediaSvc, proxyDAL, nil)
	proxyRoot := config.apiDomain + "/media/proxy/"
	mediaProxyHandler := handlers.NewMediaProxy(proxySvc, metricsRegistry.Scope("mediaproxy"))
	router.Handle("/media", mediaHandler)
	router.Handle("/media/proxy/{id}", mediaProxyHandler)
	router.Handle("/media/{id:\\w+(\\.\\w+)?}", mediaHandler)
	router.Handle("/products", handlers.NewProductsList(productsSvc, proxyRoot, proxySvc))
	router.Handle("/products/scrape", handlers.NewProductsScrape(productsSvc, proxyRoot, proxySvc))
	router.Handle("/products/{id}", handlers.NewProducts(productsSvc, proxyRoot, proxySvc))
	router.Handle("/regimen/{id}", handlers.NewRegimen(regimenSvc, mediaStore, config.webDomain, config.apiDomain))
	router.Handle("/regimen/{id}/foundation", handlers.NewFoundation(regimenSvc))
	router.Handle("/regimen/{id}/view", handlers.NewViewCount(regimenSvc))
	router.Handle("/regimen", handlers.NewRegimens(regimenSvc, mediaStore, config.webDomain, config.apiDomain))
	rxGuideHandler := handlers.NewRXGuide(rxGuideSvc)
	router.Handle(`/rxguide`, rxGuideHandler)
	router.Handle(`/rxguide/{drug_name:[A-Za-z0-9 _.,!"'/$-]+}`, rxGuideHandler)
	h := httputil.LoggingHandler(router, "regimensapi", false, requestLogger)
	h = httputil.MetricsHandler(h, metricsRegistry.Scope("regimensapi"))
	h = httputil.RequestIDHandler(h)
	h = httputil.CompressResponse(httputil.DecompressRequest(h))

	if config.corsAllowAll {
		h = httputil.ToContextHandler(cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{httputil.Delete, httputil.Get, httputil.Options, httputil.Patch, httputil.Post, httputil.Put},
			AllowCredentials: true,
			AllowedHeaders:   []string{"*"},
		}).Handler(httputil.FromContextHandler(h)))
	}
	return router, h
}

func serve(handler httputil.ContextHandler) {
	listener, err := net.Listen("tcp", config.httpAddr)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	if config.proxyProtocol {
		listener = &proxyproto.Listener{Listener: listener}
	}
	s := &http.Server{
		Handler:        httputil.FromContextHandler(handler),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	golog.Infof("Starting listener on %s...", config.httpAddr)
	golog.Fatalf(s.Serve(listener).Error())
}

func getAnalyticsLogger(metricsRegistry metrics.Registry) analytics.Logger {
	if config.analyticsDebug {
		return analytics.DebugLogger{Logf: func(s string, i ...interface{}) { golog.Infof(s, i...) }}
	} else if config.analyticsFirehoseStreams != "" {
		streams := make(map[string]string)
		for _, c := range strings.Split(config.analyticsFirehoseStreams, ",") {
			ix := strings.IndexByte(c, ':')
			if ix < 0 {
				golog.Fatalf("Analytics firehose stream flag must be in the format 'category:stream,category:stream,...'")
			}
			streams[c[:ix]] = c[ix+1:]
		}
		fh := firehose.New(awsSession())
		l, err := analytics.NewFirehoseLogger(fh, streams, config.analyticsFirehoseMaxBatchSize, config.analyticsFirehoseMaxBatchDuration, metricsRegistry.Scope("firehose"))
		if err != nil {
			golog.Fatalf("Failed to initialized firehose analytics logger: %s", err)
		}
		if err := l.Start(); err != nil {
			golog.Fatalf("Failed to start firehose analytics logger: %s", err)
		}
		return l
	} else if config.analyticsLogPath == "" {
		return &analytics.NullLogger{}
	}
	alog, err := analytics.NewFileLogger(applicationName, config.analyticsLogPath, config.analyticsMaxEvents, analytics.DefaultMaxFileAge)
	if err != nil {
		golog.Fatalf("Error while initializing analytics logger: %s", err)
	}
	if err := alog.Start(); err != nil {
		golog.Fatalf("Error while starting analytics logger: %s", err)
	}
	return alog
}

var (
	awsSess     *session.Session
	awsSessOnce sync.Once
)

// TODO: Localize this code and the client generation somewhere outside of main.go
func awsSession() *session.Session {
	awsSessOnce.Do(func() {
		var creds *credentials.Credentials
		if config.awsAccessKey != "" && config.awsSecretKey != "" {
			creds = credentials.NewStaticCredentials(config.awsAccessKey, config.awsSecretKey, config.awsToken)
		} else {
			creds = credentials.NewEnvCredentials()
			if v, err := creds.Get(); err != nil || v.AccessKeyID == "" || v.SecretAccessKey == "" {
				creds = ec2rolecreds.NewCredentials(session.New(), func(p *ec2rolecreds.EC2RoleProvider) {
					p.ExpiryWindow = time.Minute * 5
				})
			}
		}
		awsSess = session.New(&aws.Config{Region: ptr.String("us-east-1"), Credentials: creds})
	})
	return awsSess
}

func getMediaStore(cnf mediaConfig, subStore string) storage.DeterministicStore {
	switch strings.ToLower(cnf.storageType) {
	default:
		log.Fatalf("Unknown media storage type %s", cnf.storageType)
	case "s3":
		prefix := cnf.s3Prefix
		if subStore != "" {
			prefix = strings.Replace(prefix+"/cache", "//", "/", -1)
		}
		store := storage.NewS3(awsSession(), cnf.s3Bucket, prefix)
		return store
	case "local":
		pth := cnf.localStoragePath
		if subStore != "" {
			pth = path.Join(pth, "cache")
		}
		store, err := storage.NewLocalStore(pth)
		if err != nil {
			log.Fatalf("Failed to create local media store: %s", err)
		}
		return store.(storage.DeterministicStore)
	case "memory":
		return storage.NewTestStore(nil)
	}
	log.Fatal("Failed to determine a media store")
	return nil
}
