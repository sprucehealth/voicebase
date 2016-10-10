package main

import (
	"flag"
	"net/http"
	"os"
	"path"

	"github.com/rs/cors"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/google"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/handlers/auth"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/handlers/gql"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/handlers/schema"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/shttputil"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/settings"
)

var (
	flagAPIDomain       = flag.String("api_domain", "", "API `domain`")
	flagAuthTokenSecret = flag.String("auth_token_secret", "super_secret", "Auth token secret")
	flagBehindProxy     = flag.Bool("behind_proxy", false, "Flag to indicate when the service is behind a proxy")
	flagCertCacheURL    = flag.String("cert_cache_url", "", "URL path where to store cert cache (e.g. s3://bucket/path/)")
	flagLetsEncrypt     = flag.Bool("letsencrypt", false, "Enable Let's Encrypt certificates")
	flagListenAddr      = flag.String("graphql_listen_addr", "127.0.0.1:8084", "host:port to listen on")
	flagNoSSL           = flag.Bool("no_ssl", false, "Flag to force no ssl")
	flagProxyProtocol   = flag.Bool("proxy_protocol", false, "If behind a TCP proxy and proxy protocol wrapping is enabled")
	flagResourcePath    = flag.String("resource_path", path.Join(os.Getenv("GOPATH"),
		"src/github.com/sprucehealth/backend/cmd/svc/admin/resources"), "Path to resources (defaults to use GOPATH)")
	flagWebDomain = flag.String("web_domain", "localhost", "Web `domain`")

	// Services
	flagDirectoryAddr = flag.String("directory_addr", "_directory._tcp.service", "Address of the directory server")
	flagPaymentsAddr  = flag.String("payments_addr", "_payments._tcp.service", "Address of the payments server")
	flagSettingsAddr  = flag.String("settings_addr", "_settings._tcp.service", "Address of the settings server")
)

func main() {
	svc := boot.NewService("admin", nil)

	ap := google.NewAuthenticationProvider()
	conn, err := boot.DialGRPC("admin", *flagDirectoryAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	dirCli := directory.NewDirectoryClient(conn)
	conn, err = boot.DialGRPC("admin", *flagSettingsAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to settings service: %s", err)
	}
	settingsCli := settings.NewSettingsClient(conn)
	conn, err = boot.DialGRPC("admin", *flagPaymentsAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to payments service: %s", err)
	}
	paymentsCli := payments.NewPaymentsClient(conn)
	signer, err := sig.NewSigner([][]byte{[]byte(*flagAuthTokenSecret)}, nil)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	r := mux.NewRouter()
	proto := "https://"
	if *flagNoSSL {
		proto = "http://"
	}
	gqlHandler, gqlSchema := gql.New(dirCli, settingsCli, paymentsCli, signer, *flagBehindProxy)
	r.Handle("/graphql", cors.New(cors.Options{
		AllowedOrigins:   []string{proto + *flagWebDomain},
		AllowedMethods:   []string{httputil.Get, httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(
		httputil.RequestIDHandler(auth.NewAuthenticated(gqlHandler, signer))))
	r.Handle("/authenticate", cors.New(cors.Options{
		AllowedOrigins:   []string{proto + *flagWebDomain},
		AllowedMethods:   []string{httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(
		httputil.RequestIDHandler(auth.NewAuthentication(ap, signer))))
	r.Handle("/unauthenticate", cors.New(cors.Options{
		AllowedOrigins:   []string{proto + *flagWebDomain},
		AllowedMethods:   []string{httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(
		httputil.RequestIDHandler(auth.NewUnauthentication())))

	golog.Debugf("Resource path %s", *flagResourcePath)
	if !environment.IsProd() {
		if *flagResourcePath != "" {
			r.PathPrefix("/graphiql").Handler(httputil.FileServer(http.Dir(*flagResourcePath)))
		}
		r.Handle("/schema", schema.New(gqlSchema))
	}

	h := shttputil.CompressResponse(r, httputil.CompressResponse)
	h = httputil.LoggingHandler(h, "admin", *flagBehindProxy, nil)
	h = httputil.RequestIDHandler(h)

	if *flagNoSSL {
		server := &http.Server{
			Addr:           *flagListenAddr,
			Handler:        h,
			MaxHeaderBytes: 1 << 20,
		}
		go func() {
			golog.Infof("GraphQL server listening at %s...", *flagListenAddr)
			if err := server.ListenAndServe(); err != nil {
				golog.Fatalf(err.Error())
			}
		}()
	} else {
		server := &http.Server{
			Addr:      *flagListenAddr,
			TLSConfig: boot.TLSConfig(),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "https")
				h.ServeHTTP(w, r)
			}),
			MaxHeaderBytes: 1 << 20,
		}
		if !*flagLetsEncrypt {
			certs, err := boot.SelfSignedCertificate()
			if err != nil {
				golog.Fatalf("Failed to generate self signed cert %s", err)
			}
			server.TLSConfig.Certificates = certs
		} else {
			certStore, err := svc.StoreFromURL(*flagCertCacheURL)
			if err != nil {
				golog.Fatalf("Failed to generate cert cache store from url '%s': %s", *flagCertCacheURL, err)
			}
			server.TLSConfig.GetCertificate = boot.LetsEncryptCertManager(certStore.(storage.DeterministicStore), []string{*flagAPIDomain})
		}
		go func() {
			golog.Infof("GraphQL server with SSL listening at %s...", *flagListenAddr)
			if err := boot.HTTPSListenAndServe(server, *flagProxyProtocol); err != nil {
				golog.Fatalf(err.Error())
			}
		}()
	}
	golog.Debugf("Service started and waiting for termination")
	boot.WaitForTermination()
}
