package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/server"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/source/hint"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/worker"
	psettings "github.com/sprucehealth/backend/cmd/svc/patientsync/settings"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	hintlib "github.com/sprucehealth/go-hint"
	"github.com/sprucehealth/go-proxy-protocol/proxyproto"
)

var (
	flagHTTPListenAddr = flag.String("http_listen_addr", ":5001", "`host:port to listen for http requests")
	flagRPCListenAddr  = flag.String("rpc_listen_addr", ":5000", "`host:port to listen for rpc requests")
	flagBehindProxy    = flag.Bool("behind_proxy", false, "Set to true if behind a proxy")
	flagProxyProtocol  = flag.Bool("proxyproto", false, "enable proxy protocol")

	// Services
	flagDirectoryAddr = flag.String("directory_addr", "_directory._tcp.service", "host:port of directory service")
	flagThreadingAddr = flag.String("threading_addr", "_threading._tcp.service", "host:port of threading service")
	flagSettingsAddr  = flag.String("settings_addr", "_settings._tcp.service", "host:port of settings service")
	flagInviteAddr    = flag.String("invite_addr", "_invite._tcp.service", "host:port of invite service")

	// database
	flagDBHost     = flag.String("db_host", "", "database host")
	flagDBPassword = flag.String("db_password", "", "database password")
	flagDBName     = flag.String("db_name", "", "database name")
	flagDBUsername = flag.String("db_username", "", "database username")
	flagDBPort     = flag.Int("db_port", 3306, "datbase port")
	flagDBCACert   = flag.String("db_ca_cert", "", "Path to database CA certificate")
	flagDBTLS      = flag.String("db_tls", "skip-verify", " Enable TLS for database connection (one of 'true', 'false', 'skip-verify'). Ignored if CA cert provided.")

	// Encryption
	flagKMSKeyArn = flag.String("kms_key_arn", "", "arn of the master key used to encrypt/decrypt queued data")

	// Messages
	flagSyncEventQueueURL   = flag.String("sqs_sync_event_url", "", "sqs url for patient sync events")
	flagInitialSyncQueueURL = flag.String("sqs_initiate_sync_url", "", "sqs url for initiating patient sync")

	// domains
	flagWebDomain = flag.String("web_domain", "", "web domain")

	flagHintPartnerAPIKey = flag.String("hint_partner_api_key", "", "partner API key for Hint")

	svcName = "patientsync"
)

func main() {
	bootSvc := boot.NewService(svcName, nil)

	directoryConn, err := bootSvc.DialGRPC(*flagDirectoryAddr)
	if err != nil {
		golog.Fatalf("Unable to communicate with directory service: %s", err)
	}
	defer directoryConn.Close()
	directoryCLI := directory.NewDirectoryClient(directoryConn)

	threadingConn, err := bootSvc.DialGRPC(*flagThreadingAddr)
	if err != nil {
		golog.Fatalf("Unable to communicate with threading service: %s", err)
	}
	defer threadingConn.Close()
	threadingCLI := threading.NewThreadsClient(threadingConn)

	settingsConn, err := bootSvc.DialGRPC(*flagSettingsAddr)
	if err != nil {
		golog.Fatalf("Unable to communicate with settings service: %s", err)
	}
	defer settingsConn.Close()

	inviteConn, err := bootSvc.DialGRPC(*flagInviteAddr)
	if err != nil {
		golog.Fatalf("Unable to communicate with settings service: %s", err)
	}
	defer inviteConn.Close()

	awsSession, err := bootSvc.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	if *flagHintPartnerAPIKey == "" {
		golog.Fatalf("Hint PartnerAPIKey not configured")
	}
	hintlib.Key = *flagHintPartnerAPIKey
	hintlib.Testing = !environment.IsProd()

	settingsClient := settings.NewSettingsClient(settingsConn)
	inviteClient := invite.NewInviteClient(inviteConn)

	eSQS, err := awsutil.NewEncryptedSQS(*flagKMSKeyArn, kms.New(awsSession), sqs.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize sqs: %s", err)
	}

	eSNS, err := awsutil.NewEncryptedSNS(*flagKMSKeyArn, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize sns: %s", err)
	}

	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		User:          *flagDBUsername,
		Password:      *flagDBPassword,
		Host:          *flagDBHost,
		Port:          *flagDBPort,
		Name:          *flagDBName,
		CACert:        *flagDBCACert,
		EnableTLS:     *flagDBTLS == "true" || *flagDBTLS == "skip-verify",
		SkipVerifyTLS: *flagDBTLS == "skip-verify",
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}
	dl := dal.New(db)

	syncEventWorker := worker.NewSyncEvent(
		dl,
		directoryCLI,
		threadingCLI,
		settingsClient,
		inviteClient,
		eSQS,
		*flagSyncEventQueueURL,
		*flagWebDomain)
	syncEventWorker.Start()

	initiateSyncWorker := worker.NewInitiateSync(
		dl,
		*flagSyncEventQueueURL,
		*flagInitialSyncQueueURL,
		eSQS)
	initiateSyncWorker.Start()

	// start the RPC server and listen on specified port
	lis, err := net.Listen("tcp", *flagRPCListenAddr)
	if err != nil {
		golog.Fatalf("failed to listen: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = settings.RegisterConfigs(
		ctx,
		settingsClient,
		[]*settings.Config{
			psettings.ThreadTypeOptionConfig,
		})
	if err != nil {
		golog.Fatalf("Unable to register configs with the settings service: %s", err.Error())
	}
	cancel()

	subscriber, err := events.NewSQSSubscriber(eSQS, eSNS, awsSession, "patientsync")
	if err != nil {
		golog.Fatalf("unable to create subscriber: %s", err)
	}

	subscriber.Subscribe(
		"patientsync-events",
		[]events.Unmarshaler{&directory.EntityUpdatedEvent{}},
		func(u events.Unmarshaler) error {
			return worker.SyncEntityUpdate(directoryCLI, dl, u.(*directory.EntityUpdatedEvent))
		})

	srvMetricsRegistry := bootSvc.MetricsRegistry.Scope("server")
	srv := server.New(dal.New(db), settingsClient, *flagInitialSyncQueueURL, eSQS)
	patientsync.InitMetrics(srv, srvMetricsRegistry)

	s := bootSvc.GRPCServer()
	patientsync.RegisterPatientSyncServer(s, srv)
	golog.Infof("PatientSync RPC listening on %s...", *flagRPCListenAddr)
	go s.Serve(lis)

	router := mux.NewRouter().StrictSlash(true)
	router.Handle("/hint/webhook", hint.NewWebhookHandler(dal.New(db), settingsClient, *flagSyncEventQueueURL, eSQS))

	h := httputil.LoggingHandler(router, "patientsyncaapi", *flagBehindProxy, nil)
	h = httputil.RequestIDHandler(h)
	h = httputil.CompressResponse(httputil.DecompressRequest(h))
	go serve(h)

	boot.WaitForTermination()
	syncEventWorker.Shutdown()
	initiateSyncWorker.Shutdown()
	subscriber.Stop()
}

func serve(handler http.Handler) {
	listener, err := net.Listen("tcp", *flagHTTPListenAddr)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	if *flagProxyProtocol {
		listener = &proxyproto.Listener{Listener: listener}
	}
	s := &http.Server{
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	golog.Infof("Starting listener on %s...", *flagHTTPListenAddr)
	golog.Fatalf(s.Serve(listener).Error())
}
