package main

import (
	"flag"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/operational/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/operational/internal/support"
	"github.com/sprucehealth/backend/cmd/svc/operational/internal/worker"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"

	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

var (
	flagKMSKeyARN                 = flag.String("kms_key_arn", "", "the arn of the master key that should be used for encrypting data")
	flagAuthAddr                  = flag.String("auth_addr", "", "`host:port` of the auth service")
	flagExcommsAddr               = flag.String("excomms_addr", "", "`host:port` of the excomms service")
	flagDirectoryAddr             = flag.String("directory_addr", "", "`host:port` of the directory service")
	flagThreadingAddr             = flag.String("threading_addr", "", "`host:port` of the threading service")
	flagBlockAccountSQSURL        = flag.String("block_account_sqs_url", "", "url of the sqs queue for block account requests")
	flagDBName                    = flag.String("db_name", "threading", "Database name")
	flagDBHost                    = flag.String("db_host", "127.0.0.1", "Database host")
	flagDBPort                    = flag.Int("db_port", 3306, "Database port")
	flagDBUser                    = flag.String("db_user", "", "Database username")
	flagDBPass                    = flag.String("db_pass", "", "Database password")
	flagDBCACert                  = flag.String("db_ca_cert", "", "Path to database CA certificate")
	flagDBTLS                     = flag.String("db_tls", "false", "Enable TLS for database connection (one of 'true', 'false', 'skip-verify'). Ignored if CA cert provided.")
	flagSpruceOrgID               = flag.String("spruce_org_id", "", "`ID` for the Spruce support organization")
	flagOrgEventOperationalSQSURL = flag.String("org_event_sqs_url", "", "url of the sqs queue for org related events")
)

func main() {
	bootSvc := boot.NewService("operational")

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

	if *flagBlockAccountSQSURL == "" {
		golog.Fatalf("SQS URL for blocked accounts not specified")
	}

	eSQS, err := awsutil.NewEncryptedSQS(*flagKMSKeyARN, kms.New(awsSession), sqs.New(awsSession))
	if err != nil {
		golog.Fatalf(err.Error())
	}

	// Configure auth client
	if *flagAuthAddr == "" {
		golog.Fatalf("Auth service not configured")
	}
	conn, err := grpc.Dial(*flagAuthAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf(err.Error())
	}
	authClient := auth.NewAuthClient(conn)

	// Configure threading client
	if *flagThreadingAddr == "" {
		golog.Fatalf("Threading service not configured")
	}
	conn, err = grpc.Dial(*flagThreadingAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf(err.Error())
	}
	threadingClient := threading.NewThreadsClient(conn)

	// Configure excomms client
	if *flagExcommsAddr == "" {
		golog.Fatalf("Excomms service not configured")
	}
	conn, err = grpc.Dial(*flagExcommsAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf(err.Error())
	}
	excommsClient := excomms.NewExCommsClient(conn)

	// Configure directory client
	if *flagDirectoryAddr == "" {
		golog.Fatalf("Directory service not configured")
	}
	conn, err = grpc.Dial(*flagDirectoryAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf(err.Error())
	}
	directoryClient := directory.NewDirectoryClient(conn)

	if *flagSpruceOrgID == "" {
		golog.Fatalf("Spruce Org ID not specified")
	}

	w := worker.NewBlockAccountWorker(
		authClient,
		directoryClient,
		excommsClient,
		threadingClient,
		eSQS,
		dal.NewDAL(db),
		*flagBlockAccountSQSURL,
		*flagSpruceOrgID)
	w.Start()

	if *flagOrgEventOperationalSQSURL == "" {
		golog.Fatalf("SQS URL for org events required")
	}

	s := support.NewWorker(
		eSQS,
		threadingClient,
		directoryClient,
		*flagOrgEventOperationalSQSURL,
	)
	s.Start()

	boot.WaitForTermination()
}
