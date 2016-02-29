package main

import (
	"flag"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/onboarding"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/server"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification"
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
	flagSQSThreadingURL    = flag.String("sqs_threading_url", "", "SQS URL for threading queue")
	flagDirectoryAddr      = flag.String("directory_addr", "", "host:port of directory service")
	flagWebDomain          = flag.String("web_domain", "", "Domain of the website")
	flagKMSKeyARN          = flag.String("kms_key_arn", "", "the arn of the master key that should be used to encrypt outbound and decrypt inbound data")
)

func init() {
	// Disable the built in grpc tracing and use our own
	grpc.EnableTracing = false
}

func main() {
	boot.InitService("threading")

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

	awsSession, err := boot.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	eSNS, err := awsutil.NewEncryptedSNS(*flagKMSKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
		return
	}
	eSQS, err := awsutil.NewEncryptedSQS(*flagKMSKeyARN, kms.New(awsSession), sqs.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sqs: %s", err.Error())
		return
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

	if *flagDirectoryAddr == "" {
		golog.Fatalf("Directory service not configured")
	}
	conn, err := grpc.Dial(*flagDirectoryAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	directoryClient := directory.NewDirectoryClient(conn)

	dl := dal.New(db)

	w := onboarding.NewWorker(eSQS, dl, *flagWebDomain, *flagSQSEventsURL, *flagSQSThreadingURL)
	w.Start()
	defer w.Stop(time.Second * 10)

	s := grpc.NewServer()
	threading.RegisterThreadsServer(s, server.NewThreadsServer(clock.New(), dl, eSNS, *flagSNSTopicARN, notificationClient, directoryClient, *flagWebDomain))
	golog.Infof("Starting Threads service on %s...", *flagListen)

	ln, err := net.Listen("tcp", *flagListen)
	if err != nil {
		golog.Fatalf("failed to listen on %s: %v", *flagListen, err)
	}
	go func() {
		s.Serve(ln)
	}()

	boot.WaitForTermination()
}
