package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/rainycape/memcache"
	"github.com/rs/cors"
	"github.com/samuel/go-metrics/metrics"
	"github.com/samuel/go-metrics/reporter"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/products"
	"github.com/sprucehealth/backend/cmd/svc/regimens"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/handlers"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/factual"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mcutil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/go-proxy-protocol/proxyproto"
	"golang.org/x/net/context"
)

var config struct {
	httpAddr      string
	proxyProtocol bool
	webDomain     string
	env           string

	// Factual config
	factualKey    string
	factualSecret string

	// Media
	mediaStorageType      string
	mediaS3Bucket         string
	mediaS3Prefix         string
	mediaS3LatchedExpire  bool
	mediaLocalStoragePath string

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

	// CORS
	corsAllowAll bool
}

func init() {
	// Regimens service
	flag.StringVar(&config.httpAddr, "http", "0.0.0.0:8000", "listen for http on `host:port`")
	flag.BoolVar(&config.proxyProtocol, "proxyproto", false, "enabled proxy protocol")
	flag.StringVar(&config.authSecret, "auth.secret", "", "Secret to use in auth token generation")
	flag.StringVar(&config.webDomain, "web.domain", "", "The web domain used for link generation")
	flag.StringVar(&config.env, "env", "undefined", "`Environment`")

	// Factual
	flag.StringVar(&config.factualKey, "factual.key", "", "Factual API `key`")
	flag.StringVar(&config.factualSecret, "factual.secret", "", "Factual API `secret`")

	// Media
	flag.StringVar(&config.mediaStorageType, "media.storage.type", "local", "Storage type for regimen media")
	flag.StringVar(&config.mediaS3Bucket, "media.s3.bucket", "", "S3 Bucket for media storage")
	flag.StringVar(&config.mediaS3Prefix, "media.s3.prefix", "", "S3 path prefix for media storage")
	flag.BoolVar(&config.mediaS3LatchedExpire, "media.s3.latched.expire", true, "S3 configuration for latch expiration")
	flag.StringVar(&config.mediaLocalStoragePath, "media.local.path", "/tmp", "Local path to use when doing local media storage")

	// Memcached
	flag.StringVar(&config.mcDiscoveryHost, "mc.discovery.host", "", "ElastiCache discovery `host`")
	flag.DurationVar(&config.mcDiscoveryInterval, "mc.discovery.internal", time.Minute*5, "ElastiCache discovery `interval`")
	flag.StringVar(&config.mcHosts, "mc.hosts", "", "Comma separated list of memcached `hosts` when not using ElastiCache discovery")

	// AWS
	flag.StringVar(&config.awsDynamoDBEndpoint, "aws.dynamodb.endpoint", "", "AWS Dynamo DB API `endpoint`")
	flag.StringVar(&config.awsDynamoDBRegion, "aws.dynamodb.region", "", "AWS Dynamo DB API `region`")
	flag.BoolVar(&config.awsDynamoDBDisableSSL, "aws.dynamodb.disable.ssl", false, "Disable SSL in the AWS DynamoDB client")
	flag.StringVar(&config.awsAccessKey, "aws.access.key", "", "AWS Credentials Access Key")
	flag.StringVar(&config.awsSecretKey, "aws.secret.key", "", "AWS Credentials Secret Key")
	flag.StringVar(&config.awsToken, "aws.token", "", "AWS Credentials Token")

	// Amazon.com
	flag.StringVar(&config.amzAccessKey, "amz.access_key", "", "Access `key` for Amazon affiliate products API")
	flag.StringVar(&config.amzSecretKey, "amz.secret_key", "", "Secret `key` for Amazon affiliate products API")
	flag.StringVar(&config.amzAssociateTag, "amz.associate_tag", "", "Amazon affiliate associate tag")

	// Metrics
	flag.StringVar(&config.metricsSource, "metrics.source", "", "`Source` for metrics (e.g. hostname)")
	flag.StringVar(&config.libratoUsername, "librato.username", "", "Librato metrics `username`")
	flag.StringVar(&config.libratoToken, "librato.token", "", "Librato metrics auth `token`")

	// CORS
	flag.BoolVar(&config.corsAllowAll, "cors.allow.all", true, "Enable the * patterns on CORS")
}

func main() {
	log.SetFlags(log.Lshortfile)
	boot.ParseFlags("REGIMENS_")

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

	dispatcher := dispatch.New()
	productsSvc := products.New(factual.New(config.factualKey, config.factualSecret), amz, memcacheCli, metricsRegistry.Scope("productssvc"))
	regimenSvc, err := regimens.New(dynamodb.New(func() *aws.Config {
		dynamoConfig := &aws.Config{
			Region:      &config.awsDynamoDBRegion,
			DisableSSL:  &config.awsDynamoDBDisableSSL,
			Credentials: getAWSCredentials(),
		}
		if config.awsDynamoDBEndpoint != "" {
			golog.Infof("AWS Dynamo DB Endpoint configured as %s...", config.awsDynamoDBEndpoint)
			dynamoConfig.Endpoint = &config.awsDynamoDBEndpoint
		}
		if config.awsDynamoDBDisableSSL {
			dynamoConfig.DisableSSL = &config.awsDynamoDBDisableSSL
		}
		return dynamoConfig
	}()), config.authSecret)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	requestLogger := func(ctx context.Context, ev *httputil.RequestEvent) {
		av := &analytics.WebRequestEvent{
			Service:      "regimens",
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
	mediaHandler := handlers.NewMedia(config.webDomain, getMediaStore(), metricsRegistry.Scope("media"))
	router.Handle("/media", mediaHandler)
	router.Handle("/media/{id:[0-9]+}", mediaHandler)
	router.Handle("/products", handlers.NewProductsList(productsSvc))
	router.Handle("/products/scrape", handlers.NewProductsScrape(productsSvc))
	router.Handle("/products/{id}", handlers.NewProducts(productsSvc))
	router.Handle("/regimen/{id:r[0-9]+}", handlers.NewRegimen(regimenSvc, config.webDomain))
	router.Handle("/regimen", handlers.NewRegimens(regimenSvc, config.webDomain))
	h := httputil.LoggingHandler(router, requestLogger)
	h = httputil.MetricsHandler(h, metricsRegistry.Scope("regimensapi"))
	h = httputil.RequestIDHandler(h)
	h = httputil.CompressResponse(httputil.DecompressRequest(h))

	if config.corsAllowAll {
		h = httputil.ToContextHandler(cors.New(cors.Options{
			AllowedOrigins:   []string{},
			AllowedMethods:   []string{httputil.Delete, httputil.Get, httputil.Options, httputil.Patch, httputil.Post, httputil.Put},
			AllowCredentials: true,
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

// TODO: Localize this code and the client generation somewhere outside of main.go
func getAWSCredentials() *credentials.Credentials {
	var creds *credentials.Credentials
	if config.awsAccessKey != "" && config.awsSecretKey != "" {
		creds = credentials.NewStaticCredentials(config.awsAccessKey, config.awsSecretKey, config.awsToken)
	} else {
		creds = credentials.NewEnvCredentials()
		if v, err := creds.Get(); err != nil || v.AccessKeyID == "" || v.SecretAccessKey == "" {
			creds = ec2rolecreds.NewCredentials(ec2metadata.New(&ec2metadata.Config{
				HTTPClient: &http.Client{Timeout: 2 * time.Second},
			}), time.Minute*10)
		}
	}
	return creds
}

func getMediaStore() storage.DeterministicStore {
	switch strings.ToLower(config.mediaStorageType) {
	default:
		log.Fatalf("Unknown media storage type %s", config.mediaStorageType)
	case "s3":
		store := storage.NewS3(&aws.Config{Region: ptr.String("us-east-1"), Credentials: getAWSCredentials()}, config.mediaS3Bucket, config.mediaS3Prefix)
		store.LatchedExpire(config.mediaS3LatchedExpire)
		return store
	case "local":
		store, err := storage.NewLocalStore(config.mediaLocalStoragePath)
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
