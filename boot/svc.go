package boot

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // imported for side-effect of registering HTTP handlers
	"os"
	"os/signal"
	"sort"
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
	"github.com/sprucehealth/go-proxy-protocol/proxyproto"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"rsc.io/letsencrypt"
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
func NewService(name string) *Service {
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

// WaitForTermination waits for an INT or TERM signal.
func WaitForTermination() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-ch:
		golog.Infof("Quitting due to signal %s", sig.String())
	}
}

// TLSConfig returns a instance of tls.Config configured with strict defaults.
func TLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:               tls.VersionTLS10,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			// Do not include RC4 or 3DES
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		NextProtos: []string{"h2", "h2-14", "http/1.1"},
	}
}

// LetsEncryptCertManager returns functions that can be set for tls.Config.GetCertificate
// that uses Let's Encrypt to auto-register and refresh certs.
func LetsEncryptCertManager(cache storage.DeterministicStore, domains []string) func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	var m letsencrypt.Manager
	m.SetHosts(domains)

	sort.Strings(domains)
	cacheFilename := strings.Join(domains, ",") + ".cert-cache"
	b, _, err := cache.Get(cache.IDFromName(cacheFilename))
	if err != nil {
		if errors.Cause(err) != storage.ErrNoObject {
			golog.Errorf("Failed to load cert cache '%s': %s", cacheFilename, err)
		}
	} else {
		if err := m.Unmarshal(string(b)); err != nil {
			golog.Errorf("Failed to unmarshal cert cache: %s", err)
		}
	}

	go func() {
		for range m.Watch() {
			golog.Infof("Saving cert state")
			state := m.Marshal()
			if _, err := cache.Put(cacheFilename, []byte(state), "application/binary", nil); err != nil {
				golog.Errorf("Failed to write cert cache: %s", err)
			}
		}
	}()

	return m.GetCertificate
}

// HTTPSListenAndServe is a replacement for srv.ListenAndServe that
// includes optional proxy protocol support.
func HTTPSListenAndServe(srv *http.Server, proxyProtocol bool) error {
	conn, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return errors.Trace(err)
	}
	conn = tcpKeepAliveListener{conn.(*net.TCPListener)}
	if proxyProtocol {
		conn = &proxyproto.Listener{Listener: conn}
	}
	return srv.Serve(tls.NewListener(conn, srv.TLSConfig))
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away. (borrowed from net/http)
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
