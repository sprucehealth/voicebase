package main

import (
	"context"
	"flag"
	"time"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/routing/internal"
	"github.com/sprucehealth/backend/cmd/svc/routing/internal/dal"
	rsettings "github.com/sprucehealth/backend/cmd/svc/routing/internal/settings"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

var config struct {
	directoryServiceURL   string
	threadServiceURL      string
	excommsServiceURL     string
	externalMessageQueue  string
	inAppMessageQueue     string
	kmsKeyARN             string
	blockAccountsTopicARN string
	settingsServiceURL    string
	webDomain             string
	storageBucket         string
	dbHost                string
	dbPort                int
	dbPassword            string
	dbCACert              string
	dbTLS                 string
	dbUserName            string
	dbName                string
}

func init() {
	// Services
	flag.StringVar(&config.directoryServiceURL, "directory_addr", "_directory._tcp.service", "`host:port` to connect to directory service")
	flag.StringVar(&config.threadServiceURL, "threading_addr", "_threading._tcp.service", "`host:port`to connect to threading service")
	flag.StringVar(&config.excommsServiceURL, "excomms_addr", "_excomms._tcp.service", "`host:port` to connect to excomms service")
	flag.StringVar(&config.settingsServiceURL, "settings_addr", "_settings._tcp.service", "`host:port` to connect to settings service")

	// database
	flag.StringVar(&config.dbHost, "db_host", "", "database host")
	flag.StringVar(&config.dbPassword, "db_password", "", "database password")
	flag.StringVar(&config.dbName, "db_name", "", "database name")
	flag.StringVar(&config.dbUserName, "db_username", "", "database username")
	flag.IntVar(&config.dbPort, "db_port", 3306, "database port")
	flag.StringVar(&config.dbCACert, "db_ca_cert", "", "Path to database CA certificate")
	flag.StringVar(&config.dbTLS, "db_tls", "skip-verify", "Enable TLS for database connection (one of 'true', 'false', 'skip-verify'). Ignored if CA cert provided.")

	flag.StringVar(&config.externalMessageQueue, "queue_external_message", "", "queue name for receiving external messages")
	flag.StringVar(&config.inAppMessageQueue, "queue_inapp_message", "", "queue name for receiving in app messages")
	flag.StringVar(&config.kmsKeyARN, "kms_key_arn", "", "the arn of the master key that should be used to encrypt outbound and decrypt inbound data")
	flag.StringVar(&config.blockAccountsTopicARN, "block_accounts_topic_arn", "", "arn of the block accounts sns topic")
	flag.StringVar(&config.webDomain, "web_domain", "", "the baymax webapp domain")
}

func main() {
	bootSvc := boot.NewService("routing", nil)

	directoryConn, err := bootSvc.DialGRPC(config.directoryServiceURL)
	if err != nil {
		golog.Fatalf("Unable to communicate with directory service: %s", err.Error())
	}
	defer directoryConn.Close()

	threadConn, err := bootSvc.DialGRPC(config.threadServiceURL)
	if err != nil {
		golog.Fatalf("Unable to communicate with thread service: %s", err.Error())
	}
	defer threadConn.Close()

	excommsConn, err := bootSvc.DialGRPC(config.excommsServiceURL)
	if err != nil {
		golog.Fatalf("Unable to communicate with excomms service: %s", err.Error())
	}
	defer excommsConn.Close()

	settingsConn, err := bootSvc.DialGRPC(config.settingsServiceURL)
	if err != nil {
		golog.Fatalf("Unable to communicate with settings service: %s", err.Error())
	}
	defer settingsConn.Close()
	settingsClient := settings.NewSettingsClient(settingsConn)

	// register the settings with the service
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = settings.RegisterConfigs(
		ctx,
		settingsClient,
		[]*settings.Config{
			rsettings.RevealSenderAcrossExCommsConfig,
			rsettings.ProvisionedEndpointTagsConfig,
		})
	if err != nil {
		golog.Fatalf("Unable to register configs with the settings service: %s", err.Error())
	}
	cancel()

	awsSession, err := bootSvc.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	eSNS, err := awsutil.NewEncryptedSNS(config.kmsKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
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

	routingService, err := internal.NewRoutingService(
		awsSession,
		config.externalMessageQueue,
		config.inAppMessageQueue,
		directory.NewDirectoryClient(directoryConn),
		threading.NewThreadsClient(threadConn),
		excomms.NewExCommsClient(excommsConn),
		settingsClient,
		eSNS,
		config.blockAccountsTopicARN,
		config.kmsKeyARN,
		config.webDomain,
		dal.NewDAL(db),
	)
	if err != nil {
		golog.Fatalf(err.Error())
		return
	}

	golog.Infof("Started routing service ...")
	routingService.Start()

	boot.WaitForTermination()
}
