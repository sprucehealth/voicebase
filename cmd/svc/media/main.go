package main

import (
	"flag"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/handlers"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/server"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	lmedia "github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/shttputil"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/threading"
)

var (
	flagHTTPListenAddr     = flag.String("http_listen_addr", ":8081", "host:port to listen on for http requests")
	flagRPCListenAddr      = flag.String("rpc_listen_addr", ":50060", "host:port to listen on for rpc requests")
	flagWebDomain          = flag.String("web_domain", "", "Web `domain`")
	flagMediaAPIDomain     = flag.String("media_api_domain", "", "Media API `domain`")
	flagMediaStorageBucket = flag.String("media_storage_bucket", "", "storage bucket for media")
	flagSigKeys            = flag.String("signature_keys_csv", "", "csv signature keys")
	flagBehindProxy        = flag.Bool("behind_proxy", false, "Flag to indicate when the service is behind a proxy")
	flagMaxMemory          = flag.Int64("max_memory", 8<<20, "Maximum memory to use when decoding POST bodies")

	// Services
	flagAuthAddr      = flag.String("auth_addr", "_auth._tcp.service", "host:port of auth service")
	flagDirectoryAddr = flag.String("directory_addr", "_directory._tcp.service", "host:port of directory service")
	flagThreadingAddr = flag.String("threading_addr", "_threading._tcp.service", "host:port of threading service")
	flagCareAddr      = flag.String("care_addr", "_care._tcp.service", "host:port of the care service")

	// DB
	flagDBHost     = flag.String("db_host", "localhost", "the host at which we should attempt to connect to the database")
	flagDBPort     = flag.Int("db_port", 3306, "the port on which we should attempt to connect to the database")
	flagDBName     = flag.String("db_name", "media", "the name of the database which we should connect to")
	flagDBUser     = flag.String("db_user", "baymax-media", "the name of the user we should connext to the database as")
	flagDBPassword = flag.String("db_password", "baymax-media", "the password we should use when connecting to the database")
	flagDBCACert   = flag.String("db_ca_cert", "", "the ca cert to use when connecting to the database")
	flagDBTLSCert  = flag.String("db_tls_cert", "", "the tls cert to use when connecting to the database")
	flagDBTLSKey   = flag.String("db_tls_key", "", "the tls key to use when connecting to the database")
	flagDBTLS      = flag.String("db_tls", "skip-verify", "Enable TLS for database connection (one of 'true', 'false', 'skip-verify'). Ignored if CA cert provided.")
)

func main() {
	svc := boot.NewService("media", nil)
	awsSession, err := svc.AWSSession()
	if err != nil {
		golog.Fatalf("Failed to create AWS session: %s", err)
	}

	if *flagMediaAPIDomain == "" {
		golog.Fatalf("Media API Domain not specified")
	}

	if *flagMediaStorageBucket == "" {
		golog.Fatalf("Media Storage bucket not specified")
	}

	conn, err := svc.DialGRPC(*flagAuthAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to auth service: %s", err)
	}
	authClient := auth.NewAuthClient(conn)

	conn, err = svc.DialGRPC(*flagDirectoryAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	directoryClient := directory.NewDirectoryClient(conn)

	conn, err = svc.DialGRPC(*flagThreadingAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to threading service: %s", err)
	}
	threadingClient := threading.NewThreadsClient(conn)

	conn, err = svc.DialGRPC(*flagCareAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to care service :%s", err)
	}
	careClient := care.NewCareClient(conn)

	if *flagRPCListenAddr == "" {
		golog.Fatalf("RPC listen addr required")
	}
	lis, err := net.Listen("tcp", *flagRPCListenAddr)
	if err != nil {
		golog.Fatalf("failed to listen: %v", err)
	}

	golog.Infof("Initializing database connection on %s:%d, user: %s, db: %s...", *flagDBHost, *flagDBPort, *flagDBUser, *flagDBName)
	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:          *flagDBHost,
		Port:          *flagDBPort,
		Name:          *flagDBName,
		User:          *flagDBUser,
		Password:      *flagDBPassword,
		CACert:        *flagDBCACert,
		TLSCert:       *flagDBTLSCert,
		TLSKey:        *flagDBTLSKey,
		EnableTLS:     *flagDBTLS == "true" || *flagDBTLS == "skip-verify",
		SkipVerifyTLS: *flagDBTLS == "skip-verify",
	})
	if err != nil {
		golog.Fatalf("Failed to initialize DB connection: %s", err)
	}

	if *flagSigKeys == "" {
		golog.Fatalf("signature_keys_csv required")
	}
	sigKeys := strings.Split(*flagSigKeys, ",")
	sigKeysByteSlice := make([][]byte, len(sigKeys))
	for i, sk := range sigKeys {
		sigKeysByteSlice[i] = []byte(sk)
	}
	signer, err := sig.NewSigner(sigKeysByteSlice, nil)
	if err != nil {
		golog.Fatalf("Failed to create signer: %s", err.Error())
	}

	dl := dal.New(db)
	msvc := initService(awsSession, dl, directoryClient, threadingClient, careClient, *flagMediaStorageBucket)

	r := mux.NewRouter()
	handlers.InitRoutes(
		r,
		msvc,
		authClient,
		urlutil.NewSigner("https://"+*flagMediaAPIDomain, signer, clock.New()),
		*flagWebDomain,
		*flagMediaAPIDomain,
		*flagMaxMemory)
	h := httputil.LoggingHandler(r, "media", *flagBehindProxy, nil)

	golog.Infof("Media HTTP Listening on %s...", *flagHTTPListenAddr)
	httpSrv := &http.Server{
		Addr:           *flagHTTPListenAddr,
		Handler:        shttputil.CompressResponse(h, httputil.CompressResponse),
		MaxHeaderBytes: 1 << 20,
		// 5 minute timeout is to allow for media uploads/downloads
		ReadTimeout: 5 * time.Minute,
		// Note: reason we don't include a write timeout is because we redirect the client
		// to get media objects directly from S3.
	}
	go func() {
		httpSrv.ListenAndServe()
	}()

	srvMetricsRegistry := svc.MetricsRegistry.Scope("server")
	srv := server.New(dl, msvc, *flagMediaAPIDomain)
	media.InitMetrics(srv, srvMetricsRegistry)
	s := svc.GRPCServer()
	media.RegisterMediaServer(s, srv)
	golog.Infof("Media RPC listening on %s...", *flagRPCListenAddr)
	go s.Serve(lis)

	boot.WaitForTermination()
	svc.Shutdown()
}

func initService(
	awsSession *session.Session,
	dal dal.DAL,
	directoryClient directory.DirectoryClient,
	threadingClient threading.ThreadsClient,
	careClient care.CareClient,
	mediaStorageBucket string) service.Service {
	s3Store := storage.NewS3(awsSession, mediaStorageBucket, "media")
	s3CacheStore := storage.NewS3(awsSession, mediaStorageBucket, "media-cache")
	return service.New(
		dal,
		directoryClient,
		threadingClient,
		careClient,
		lmedia.NewImageService(s3Store, s3CacheStore, 0, 0),
		lmedia.NewAudioService(s3Store, s3CacheStore, 0),
		lmedia.NewVideoService(s3Store, s3CacheStore, 0),
		lmedia.NewBinaryService(s3Store, s3CacheStore, 0),
	)
}
