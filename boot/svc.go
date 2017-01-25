package boot

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
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
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/grpcdns"
	"github.com/sprucehealth/backend/libs/mcutil"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/smet"
	"github.com/sprucehealth/backend/libs/storage"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

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
		segmentIOKey           string
		jsonLogs               bool
		tlsClients             bool
		tlsCACertPath          string
		tlsCertPath            string
		tlsKeyPath             string
	}
	name                string
	awsSessionOnce      sync.Once
	awsSession          *session.Session
	awsSessionErr       error
	memcacheOnce        sync.Once
	memcacheCli         *memcache.Client
	memcacheErr         error
	healthServerOnce    sync.Once
	healthServer        *health.Server
	grpcServerOnce      sync.Once
	grpcServer          *grpc.Server
	grpcServerTLSConfig *tls.Config
	grpcClientTLSConfig *tls.Config
	clientCreds         credentials.TransportCredentials
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
	flag.StringVar(&svc.flags.segmentIOKey, "segmentio_key", "", "Segment IO API `key`")
	flag.BoolVar(&svc.flags.jsonLogs, "json_logs", false, "Enable JSON formatted logs")
	flag.BoolVar(&svc.flags.tlsClients, "tls_clients", false, "Enable JSON formatted logs")
	flag.StringVar(&svc.flags.tlsCACertPath, "tls_ca_cert_path", "", "Path to TLS CA certificate")
	flag.StringVar(&svc.flags.tlsCertPath, "tls_cert_path", "", "Path to TLS certificate")
	flag.StringVar(&svc.flags.tlsKeyPath, "tls_key_path", "", "Path to TLS key")

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

	if svc.flags.tlsCertPath != "" && svc.flags.tlsKeyPath != "" {
		golog.Infof("Enabling TLS with cert %s and key %s", svc.flags.tlsCertPath, svc.flags.tlsKeyPath)
		cert, err := tls.LoadX509KeyPair(svc.flags.tlsCertPath, svc.flags.tlsKeyPath)
		if err != nil {
			golog.Fatalf("Failed to load TLS cert or key: %s", err)
		}
		svc.grpcServerTLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			// Only use curves which have assembly implementations. See https://blog.gopheracademy.com/advent-2016/exposing-go-on-the-internet/
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				// tls.X25519, // TODO: Go 1.8 only
			},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				// Do not include RC4 or 3DES
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				// tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // TODO: Go 1.8 only
				// tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // TODO: Go 1.8 only
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
			Certificates: []tls.Certificate{cert},
		}
	}

	if svc.flags.tlsCACertPath != "" {
		creds, err := credentials.NewClientTLSFromFile(svc.flags.tlsCACertPath, "")
		if err != nil {
			golog.Fatalf("Failed to load CA cert: %s", err)
		}
		svc.clientCreds = creds
	}

	if svc.flags.tlsClients {
		if svc.flags.tlsCACertPath == "" {
			golog.Infof("Using TLS for service clients")
			svc.grpcClientTLSConfig = &tls.Config{}
		} else {
			golog.Infof("Using TLS for service clients with CA %s", svc.flags.tlsCACertPath)
			cp, err := CAFromFile(svc.flags.tlsCACertPath)
			if err != nil {
				golog.Fatalf("Failed to create CA pool: %s", err.Error())
			}
			svc.grpcClientTLSConfig = &tls.Config{
				RootCAs: cp,
			}
		}
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

	if svc.flags.segmentIOKey != "" {
		analytics.InitSegment(svc.flags.segmentIOKey)
	}

	// Establish the metrics handler on management
	http.Handle("/metrics", metrics.RegistryHandler(svc.MetricsRegistry))
	smet.Init(svc.MetricsRegistry)

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

func (svc *Service) SQSURL(name string) (string, error) {
	awsSession, err := svc.AWSSession()
	if err != nil {
		return "", err
	}

	accountID, err := getAWSAccountID(awsSession)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(name, environment.GetCurrent()) {
		name = environment.GetCurrent() + "-" + name
	}

	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", *awsSession.Config.Region, accountID, name), nil
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

// HealthServer returns a singleton of the health server.
func (svc *Service) HealthServer() *health.Server {
	svc.healthServerOnce.Do(func() {
		svc.healthServer = health.NewServer()
		// Set the default to serving since it won't actually have any effect
		// until the server is listening.
		svc.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	})
	return svc.healthServer
}

// GRPCServer returns a GRPC server single with a health service registered,
// and any default options set.
func (svc *Service) GRPCServer() *grpc.Server {
	svc.grpcServerOnce.Do(func() {
		var opts []grpc.ServerOption
		if svc.grpcServerTLSConfig != nil {
			opts = append(opts, grpc.Creds(credentials.NewTLS(svc.grpcServerTLSConfig)))
		}
		svc.grpcServer = grpc.NewServer(opts...)
		grpc_health_v1.RegisterHealthServer(svc.grpcServer, svc.HealthServer())
	})
	return svc.grpcServer
}

// DialGRPC connects to a GRPC service with the given address.
func (svc *Service) DialGRPC(addr string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return DialGRPC(svc.name, addr, svc.grpcClientTLSConfig)
}

// Shutdown performs a graceful shutdown.
func (svc *Service) Shutdown() {
	svc.HealthServer().SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	svc.GRPCServer().GracefulStop()
}

// DialGRPC connects to a GRPC service with the given address. Agent is
// the name of the service making the connection (used to build the user agent).
func DialGRPC(agent, addr string, tlsConfig *tls.Config, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if addr == "" {
		return nil, errors.New("empty address")
	}
	if tlsConfig == nil {
		opts = append(opts, grpc.WithInsecure())
	} else {
		// Can't do a value copy here (tlsconfigCopy := *tls.Config) because the config contains a mutex.
		tlsConfigCopy := &tls.Config{
			Certificates:       tlsConfig.Certificates,
			NameToCertificate:  tlsConfig.NameToCertificate,
			GetCertificate:     tlsConfig.GetCertificate,
			RootCAs:            tlsConfig.RootCAs,
			ServerName:         tlsConfig.ServerName,
			InsecureSkipVerify: tlsConfig.InsecureSkipVerify,
			CipherSuites:       tlsConfig.CipherSuites,
			MinVersion:         tlsConfig.MinVersion,
			MaxVersion:         tlsConfig.MaxVersion,
		}
		if tlsConfigCopy.ServerName == "" && addr[0] == '_' {
			// Rewrite SRV hostnames since they're not valid hostnames for certificate validation
			// _servicename._tcp.service.* -> servicename.service.*
			a := strings.SplitN(addr, ".", 3)
			tlsConfigCopy.ServerName = a[0][1:] + "." + a[2]
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfigCopy)))
	}
	opts = append(opts, grpc.WithBalancer(grpc.RoundRobin(grpcdns.Resolver(time.Second*5))))
	opts = append(opts, grpc.WithUserAgent(fmt.Sprintf("%s/%s", agent, BuildNumber)))
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return conn, nil
}

// CAFromFile loads a CA certificate from the provided path and creates a new pool.
func CAFromFile(path string) (*x509.CertPool, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Errorf("Failed to read CA file %s: %s", path, err)
	}
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(b) {
		return nil, errors.Errorf("Failed to append CA certificate")
	}
	return cp, nil
}

// WaitForTermination waits for an INT or TERM signal.
func WaitForTermination() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Reset(os.Interrupt, syscall.SIGTERM)
	}()
	select {
	case sig := <-ch:
		golog.Infof("Quitting due to signal %s", sig.String())
	}
}
