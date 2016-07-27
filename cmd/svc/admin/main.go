package main

import (
	"flag"
	"net/http"
	"os"
	"path"

	"github.com/rs/cors"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/handlers/auth"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/handlers/gql"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/handlers/logging"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/handlers/schema"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/ldap"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/shttputil"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
)

var (
	flagAuthTokenSecret = flag.String("auth_token_secret", "super_secret", "Auth token secret")
	flagBehindProxy     = flag.Bool("behind_proxy", false, "Flag to indicate when the service is behind a proxy")
	flagDirectoryAddr   = flag.String("directory_addr", "127.0.0.1:50052", "Address of the directory server")
	flagLDAPAddr        = flag.String("ldap_addr", "localhost:389", "Address of the LDAP server")
	flagLDAPBaseDN      = flag.String("ldap_base_dn", "ou=People,dc=sprucehealth,dc=com", "The base DN for LDAP users")
	flagLetsEncrypt     = flag.Bool("letsencrypt", false, "Enable Let's Encrypt certificates")
	flagListenAddr      = flag.String("graphql_listen_addr", "127.0.0.1:8084", "host:port to listen on")
	flagResourcePath    = flag.String("resource_path", path.Join(os.Getenv("GOPATH"),
		"src/github.com/sprucehealth/backend/cmd/svc/admin/resources"), "Path to resources (defaults to use GOPATH)")
	flagSettingsAddr = flag.String("settings_addr", "127.0.0.1:50054", "Address of the settings server")
	flagWebDomain    = flag.String("web_domain", "localhost", "Web `domain`")
)

func main() {
	boot.NewService("admin", nil)

	ap, err := ldap.NewAuthenticationProvider(&ldap.Config{
		Address: *flagLDAPAddr,
		BaseDN:  *flagLDAPBaseDN,
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}
	conn, err := boot.DialGRPC("admin", *flagDirectoryAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	dirCli := directory.NewDirectoryClient(conn)
	conn, err = boot.DialGRPC("admin", *flagSettingsAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	settingsCli := settings.NewSettingsClient(conn)
	signer, err := sig.NewSigner([][]byte{[]byte(*flagAuthTokenSecret)}, nil)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	r := mux.NewRouter()
	gqlHandler, gqlSchema := gql.New(ap, dirCli, settingsCli, signer, *flagBehindProxy)
	r.Handle("/graphql", cors.New(cors.Options{
		AllowedOrigins:   []string{"https://" + *flagWebDomain},
		AllowedMethods:   []string{httputil.Get, httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(
		httputil.RequestIDHandler(
			logging.NewRequestID(
				auth.NewAuthenticated(gqlHandler, signer)))))
	r.Handle("/authenticate", cors.New(cors.Options{
		AllowedOrigins:   []string{"https://" + *flagWebDomain},
		AllowedMethods:   []string{httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(
		httputil.RequestIDHandler(
			logging.NewRequestID(
				auth.NewAuthentication(ap, signer)))))

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

	go func() {
		server := &http.Server{
			Addr:           *flagListenAddr,
			Handler:        h,
			MaxHeaderBytes: 1 << 20,
		}
		golog.Infof("GraphQL server listening at %s...", *flagListenAddr)
		server.ListenAndServe()
	}()
	boot.WaitForTermination()
}
