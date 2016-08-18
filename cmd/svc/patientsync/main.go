package main

import (
	"flag"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/service"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

var (
	// Services
	flagDirectoryAddr = flag.String("directory_addr", "_directory._tcp.service", "host:port of directory service")
	flagThreadingAddr = flag.String("threading_addr", "_threading._tcp.service", "host:port of threading service")

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
	flagSyncEventQueue = flag.String("sqs_emr_sync_event_url", "", "sqs url for emr sync events")

	// domains
	flagWebDomain = flag.String("web_domain", "", "web domain")

	svcName = "patientsync"
)

func main() {

	bootSvc := boot.NewService(svcName, nil)

	directoryConn, err := boot.DialGRPC(svcName, *flagDirectoryAddr)
	if err != nil {
		golog.Fatalf("Unable to communicate with directory service: %s", err)
	}
	defer directoryConn.Close()

	threadingConn, err := boot.DialGRPC(svcName, *flagThreadingAddr)
	if err != nil {
		golog.Fatalf("Unable to communicate with threading service: %s", err)
	}
	defer threadingConn.Close()

	awsSession, err := bootSvc.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	eSQS, err := awsutil.NewEncryptedSQS(*flagKMSKeyArn, kms.New(awsSession), sqs.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize sqs: %s", err)
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

	s := service.New(
		dal.New(db),
		directory.NewDirectoryClient(directoryConn),
		threading.NewThreadsClient(threadingConn),
		eSQS,
		*flagSyncEventQueue,
		*flagWebDomain)
	s.Start()

	boot.WaitForTermination()
	s.Shutdown()
}
