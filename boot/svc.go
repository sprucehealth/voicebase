package boot

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // imported for side-effect of registering HTTP handlers
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/rainycape/memcache"
	"github.com/samuel/go-metrics/metrics"
	"github.com/samuel/go-metrics/reporter"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mcutil"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/storage"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
)

type Service struct {
	MetricsRegistry metrics.Registry

	flags struct {
		debug                  bool
		env                    string
		errorSNSTopic          string
		managementAddr         string
		libratoUsername        string
		libratoToken           string
		awsAccessKey           string
		awsSecretKey           string
		awsToken               string
		awsRegion              string
		memcachedDiscoveryAddr string
		memcachedHosts         string
		jsonLogs               bool
	}
	name           string
	awsSessionOnce sync.Once
	awsSession     *session.Session
	awsSessionErr  error
	memcacheOnce   sync.Once
	memcacheCli    *memcache.Client
	memcacheErr    error
}

// NewService should be called at the start of a service. It parses flags and sets up a mangement server.
func NewService(name string, healthCheckHandler http.Handler) *Service {
	svc := &Service{name: name}
	flag.BoolVar(&svc.flags.debug, "debug", false, "Enable debug logging")
	flag.StringVar(&svc.flags.env, "env", "", "Execution environment")
	flag.StringVar(&svc.flags.errorSNSTopic, "error_sns_topic", "", "SNS `topic` which to send errors")
	flag.StringVar(&svc.flags.managementAddr, "management_addr", ":9000", "`host:port` of management HTTP server")
	flag.StringVar(&svc.flags.libratoUsername, "librato_username", "", "Librato metrics username")
	flag.StringVar(&svc.flags.libratoToken, "librato_token", "", "Librato metrics token")
	flag.StringVar(&svc.flags.awsAccessKey, "aws_access_key", "", "Access `key` for AWS")
	flag.StringVar(&svc.flags.awsSecretKey, "aws_secret_key", "", "Secret `key` for AWS")
	flag.StringVar(&svc.flags.awsToken, "aws_token", "", "Temporary access `token` for AWS")
	flag.StringVar(&svc.flags.awsRegion, "aws_region", "us-east-1", "AWS `region`")
	flag.StringVar(&svc.flags.memcachedDiscoveryAddr, "memcached_discovery_addr", "", "host:port of memcached discovery service")
	flag.StringVar(&svc.flags.memcachedHosts, "memcached_hosts", "", "Comma separate host:port list of memcached server addresses")
	flag.BoolVar(&svc.flags.jsonLogs, "json_logs", false, "Enable JSON formatted logs")

	ParseFlags(strings.ToUpper(name) + "_")

	if svc.flags.env == "" {
		golog.Fatalf("-env flag required")
	}
	environment.SetCurrent(svc.flags.env)

	if svc.flags.jsonLogs {
		golog.Default().SetHandler(golog.WriterHandler(os.Stderr, golog.JSONFormatter(true)))
	}

	if svc.flags.debug {
		golog.Default().SetLevel(golog.DEBUG)
	}

	// Use the built-in tracing for now, we'll want our own eventually to be able
	// to track cross-service traces, but this might help for now.
	grpc.EnableTracing = !environment.IsProd()
	// Have to override the default AuthRequest because due to docker we'll never
	// actually see localhost.
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		host, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			host = req.RemoteAddr
		}
		switch host {
		case "localhost", "127.0.0.1", "::1":
			return true, true
		}
		return true, true
	}

	// TODO: this can be expanded in the future to support registering custom health checks (e.g. checking connection to DB)
	http.Handle("/health-check", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if healthCheckHandler != nil {
			healthCheckHandler.ServeHTTP(w, r)
			return
		}
		w.Write([]byte("OK"))
	}))

	// Start management server
	go func() {
		golog.Fatalf("%s", http.ListenAndServe(svc.flags.managementAddr, nil))
	}()

	metricsRegistry := metrics.NewRegistry()
	svc.MetricsRegistry = metricsRegistry.Scope("svc." + name)

	if svc.flags.errorSNSTopic != "" {
		awsSession, err := svc.AWSSession()
		if err != nil {
			golog.Fatalf("Failed to create AWS session: %s", err)
		}
		var rateLimiter ratelimit.KeyedRateLimiter
		mc, err := svc.MemcacheClient()
		if err != nil {
			golog.Fatalf("Failed to create memcached client: %s", err)
		} else if mc != nil {
			rateLimiter = ratelimit.NewMemcache(mc, 5, 60)
		} else {
			rateLimiter = ratelimit.NewLRUKeyed(128, func() ratelimit.RateLimiter {
				return ratelimit.NewSimple(5, time.Minute)
			})
		}
		golog.Default().SetHandler(SNSLogHandler(
			sns.New(awsSession), svc.flags.errorSNSTopic, environment.GetCurrent()+"/"+name,
			golog.Default().Handler(), rateLimiter, metricsRegistry.Scope("errorsns")))
	}

	metricsRegistry.Add("runtime", metrics.RuntimeMetrics)

	if svc.flags.libratoUsername != "" && svc.flags.libratoToken != "" {
		source := svc.flags.env + "-" + name
		statsReporter := reporter.NewLibratoReporter(
			metricsRegistry, time.Minute, true, svc.flags.libratoUsername, svc.flags.libratoToken, source)
		statsReporter.Start()
	}

	return svc
}

// AWSSession returns an AWS session.
func (svc *Service) AWSSession() (*session.Session, error) {
	svc.awsSessionOnce.Do(func() {
		awsConfig, err := awsutil.Config(svc.flags.awsRegion, svc.flags.awsAccessKey, svc.flags.awsSecretKey, svc.flags.awsToken)
		if err != nil {
			svc.awsSessionErr = err
			return
		}
		svc.awsSession = session.New(awsConfig)
	})
	return svc.awsSession, svc.awsSessionErr
}

// MemcacheClient lazily creates and returns a memcached client. It returns the same client on every call.
func (svc *Service) MemcacheClient() (*memcache.Client, error) {
	if svc.flags.memcachedDiscoveryAddr == "" && svc.flags.memcachedHosts == "" {
		return nil, nil
	}
	svc.memcacheOnce.Do(func() {
		var servers memcache.Servers
		if svc.flags.memcachedDiscoveryAddr != "" {
			discoveryInterval := time.Minute
			d, err := awsutil.NewElastiCacheDiscoverer(svc.flags.memcachedDiscoveryAddr, discoveryInterval)
			if err != nil {
				svc.memcacheErr = fmt.Errorf("Failed to discover memcached hosts: %s", err.Error())
				return
			}
			servers = mcutil.NewElastiCacheServers(d)
		} else {
			var hosts []string
			for _, h := range strings.Split(svc.flags.memcachedHosts, ",") {
				if h = strings.TrimSpace(h); h != "" {
					hosts = append(hosts, h)
				}
			}
			if len(hosts) == 0 {
				svc.memcacheErr = fmt.Errorf("Empty memcached host list")
				return
			}
			servers = mcutil.NewHRWServer(hosts)
		}
		svc.memcacheCli = memcache.NewFromServers(servers)
	})
	return svc.memcacheCli, svc.memcacheErr
}

// StoreFromURL returns a storage.Store created based on the URL procided. The scheme represents
// the storage type (file or s3). For S3 the host reseprents the bucket.
func (svc *Service) StoreFromURL(u string) (storage.Store, error) {
	ur, err := url.Parse(u)
	if err != nil {
		return nil, errors.Errorf("failed to parse URL: %s", err)
	}
	switch ur.Scheme {
	case "file":
		return storage.NewLocalStore(ur.Path)
	case "s3":
		if ur.Host == "" {
			return nil, errors.Errorf("S3 storage URL '%s' missing bucket (aka host)", u)
		}
		awsSession, err := svc.AWSSession()
		if err != nil {
			return nil, errors.Trace(err)
		}
		return storage.NewS3(awsSession, ur.Host, strings.TrimPrefix(ur.Path, "/")), nil
	}
	return nil, errors.Errorf("no storage available for scheme %s", ur.Scheme)
}

// WaitForTermination waits for an INT or TERM signal.
func WaitForTermination() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-ch:
		golog.Infof("Quitting due to signal %s", sig.String())
	}
}
