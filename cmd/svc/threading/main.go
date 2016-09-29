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
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/server"
	tsettings "github.com/sprucehealth/backend/cmd/svc/threading/internal/settings"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/setupthread"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

var (
	flagDBName             = flag.String("db_name", "threading", "Database name")
	flagDBHost             = flag.String("db_host", "127.0.0.1", "Database host")
	flagDBPort             = flag.Int("db_port", 3306, "Database port")
	flagDBUser             = flag.String("db_user", "", "Database username")
	flagDBPass             = flag.String("db_pass", "", "Database password")
	flagDBCACert           = flag.String("db_ca_cert", "", "Path to database CA certificate")
	flagDBTLS              = flag.String("db_tls", "false", "Enable TLS for database connection (one of 'true', 'false', 'skip-verify'). Ignored if CA cert provided.")
	flagListen             = flag.String("listen_addr", ":5001", "Address on which to listen")
	flagSNSTopicARN        = flag.String("sns_topic_arn", "", "SNS topic ARN to publish new messages to")
	flagSQSNotificationURL = flag.String("sqs_notification_url", "", "the sqs url for notification messages")
	flagSQSEventsURL       = flag.String("sqs_events_url", "", "SQS URL for events queue")
	flagWebDomain          = flag.String("web_domain", "", "Domain of the website")
	flagKMSKeyARN          = flag.String("kms_key_arn", "", "the arn of the master key that should be used to encrypt outbound and decrypt inbound data")

	// Services
	flagSettingsAddr  = flag.String("settings_addr", "_settings._tcp.service", "host:port of settings service")
	flagMediaAddr     = flag.String("media_addr", "_media._tcp.service", "host:port of media service")
	flagDirectoryAddr = flag.String("directory_addr", "_directory._tcp.service", "host:port of directory service")
	flagPaymentsAddr  = flag.String("payments_addr", "_payments._tcp.service", "host:port of payments service")
)

func init() {
	// Disable the built in grpc tracing and use our own
	grpc.EnableTracing = false
}

func main() {
	bootSvc := boot.NewService("threading", nil)

	settingsConn, err := boot.DialGRPC("threading", *flagSettingsAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to settings service: %s", err)
	}
	defer settingsConn.Close()
	settingsClient := settings.NewSettingsClient(settingsConn)

	if *flagKMSKeyARN == "" {
		golog.Fatalf("-kms_key_arn flag is required")
	}

	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:          *flagDBHost,
		Port:          *flagDBPort,
		Name:          *flagDBName,
		User:          *flagDBUser,
		Password:      *flagDBPass,
		EnableTLS:     *flagDBTLS == "true" || *flagDBTLS == "skip-verify",
		SkipVerifyTLS: *flagDBTLS == "skip-verify",
		CACert:        *flagDBCACert,
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}

	awsSession, err := bootSvc.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	eSNS, err := awsutil.NewEncryptedSNS(*flagKMSKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
	}
	eSQS, err := awsutil.NewEncryptedSQS(*flagKMSKeyARN, kms.New(awsSession), sqs.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sqs: %s", err.Error())
	}

	// Start management server
	go func() {
		golog.Fatalf("%s", http.ListenAndServe(":8005", nil))
	}()

	var notificationClient notification.Client
	if *flagSQSNotificationURL != "" {
		notificationClient = notification.NewClient(eSQS, &notification.ClientConfig{
			SQSNotificationURL: *flagSQSNotificationURL,
		})
	}

	conn, err := boot.DialGRPC("threading", *flagDirectoryAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	directoryClient := directory.NewDirectoryClient(conn)

	conn, err = boot.DialGRPC("threading", *flagMediaAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to media service: %s", err)
	}
	mediaClient := media.NewMediaClient(conn)

	conn, err = boot.DialGRPC("threading", *flagPaymentsAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to payments service: %s", err)
	}
	paymentsClient := payments.NewPaymentsClient(conn)

	dl := dal.New(db)

	// register the settings with the service
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = settings.RegisterConfigs(
		ctx,
		settingsClient,
		[]*settings.Config{
			tsettings.AlertAllMessagesConfig,
			tsettings.PreviewPatientMessageContentInNotificationConfig,
			tsettings.PreviewTeamMessageContentInNotificationConfig,
		})
	if err != nil {
		golog.Fatalf("Unable to register configs with the settings service: %s", err.Error())
	}
	cancel()

	srv := server.NewThreadsServer(clock.New(), dl, eSNS, *flagSNSTopicARN, notificationClient, directoryClient, settingsClient, mediaClient, paymentsClient, *flagWebDomain)
	threading.InitMetrics(srv, bootSvc.MetricsRegistry.Scope("server"))

	w := setupthread.NewWorker(eSQS, workerClient{srv: srv}, *flagSQSEventsURL)
	w.Start()
	defer w.Stop(time.Second * 10)

	s := bootSvc.GRPCServer()
	threading.RegisterThreadsServer(s, srv)
	golog.Infof("Starting Threads service on %s...", *flagListen)

	ln, err := net.Listen("tcp", *flagListen)
	if err != nil {
		golog.Fatalf("failed to listen on %s: %v", *flagListen, err)
	}
	go s.Serve(ln)

	boot.WaitForTermination()
	bootSvc.Shutdown()
}

// workerClient allows using the server directly as a client. avoids the worker from having to make calls out and back in
// which would introduce a weird start-time dependency due to running in the same process.
type workerClient struct {
	srv threading.ThreadsServer
}

func (wc workerClient) OnboardingThreadEvent(ctx context.Context, req *threading.OnboardingThreadEventRequest, opts ...grpc.CallOption) (*threading.OnboardingThreadEventResponse, error) {
	return wc.srv.OnboardingThreadEvent(ctx, req)
}
