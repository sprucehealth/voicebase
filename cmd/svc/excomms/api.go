package main

import (
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	analytics "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/handlers"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/twilio"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/go-proxy-protocol/proxyproto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func runAPI(bootSvc *boot.Service) {
	conn, err := grpc.Dial(config.directoryServiceURL, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with events processor service: %s", err.Error())
	}
	defer conn.Close()

	awsSession, err := bootSvc.AWSSession()
	if err != nil {
		golog.Fatalf("Failed to create AWS session: %s", err)
	}

	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		User:          config.dbUserName,
		Password:      config.dbPassword,
		Host:          config.dbHost,
		Port:          config.dbPort,
		Name:          config.dbName,
		CACert:        config.dbCACert,
		EnableTLS:     config.dbTLS == "true" || config.dbTLS == "skip-verify",
		SkipVerifyTLS: config.dbTLS == "skip-verify",
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}

	dl := dal.NewDAL(db)

	store := storage.NewS3(awsSession, config.attachmentBucket, config.attachmentPrefix)
	proxyNumberManager := proxynumber.NewManager(dl, clock.New())

	settingsConn, err := grpc.Dial(
		config.settingsServiceURL,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with settings service: %s", err.Error())
		return
	}
	defer settingsConn.Close()

	eSNS, err := awsutil.NewEncryptedSNS(config.kmsKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
		return
	}

	var segmentClient *analytics.Client
	if config.segmentIOKey != "" {
		segmentClient = analytics.New(config.segmentIOKey)
		defer segmentClient.Close()
	}

	eh := twilio.NewEventHandler(
		directory.NewDirectoryClient(conn),
		settings.NewSettingsClient(settingsConn),
		dl,
		eSNS,
		clock.New(),
		proxyNumberManager,
		config.excommsAPIURL,
		config.externalMessageTopic,
		config.incomingRawMessageTopic,
		config.resourceCleanerTopic,
		segmentClient,
		storage.NewS3(awsSession, config.attachmentBucket, "media"))

	router := mux.NewRouter().StrictSlash(true)
	router.Handle("/twilio/sms", handlers.NewTwilioSMSHandler(dl, config.incomingRawMessageTopic, eSNS))
	router.Handle("/twilio/sms/status", handlers.NewTwilioSMSStatusHandler(eh))
	router.Handle("/twilio/call/{event}", handlers.NewTwilioRequestHandler(eh, bootSvc.MetricsRegistry.Scope("voice")))
	router.Handle("/sendgrid/email", handlers.NewSendGridHandler(config.incomingRawMessageTopic, eSNS, dl, store))

	webRequestLogger := func(ctx context.Context, ev *httputil.RequestEvent) {

		contextVals := []interface{}{
			"Method", ev.Request.Method,
			"URL", ev.URL.String(),
			"UserAgent", ev.Request.UserAgent(),
			"RequestID", httputil.RequestID(ctx),
			"RemoteAddr", ev.RemoteAddr,
			"StatusCode", ev.StatusCode,
		}

		log := golog.Context(contextVals...)

		if ev.Panic != nil {
			log.Criticalf("http: panic: %v\n%s", ev.Panic, ev.StackTrace)
		} else {
			log.Infof("excommsapi")
		}
	}

	h := httputil.LoggingHandler(router, webRequestLogger)
	h = httputil.RequestIDHandler(h)
	h = httputil.CompressResponse(httputil.DecompressRequest(h))
	serve(h)
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
	// TODO: Only listen on secure connection.
	golog.Fatalf(s.Serve(listener).Error())
}
