package main

import (
	"flag"
	"time"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/notification/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/notification/internal/service"
	nsettings "github.com/sprucehealth/backend/cmd/svc/notification/internal/settings"
	cfg "github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var config struct {
	dbHost                            string
	dbPort                            int
	dbName                            string
	dbUser                            string
	dbPassword                        string
	dbCACert                          string
	dbTLSCert                         string
	dbTLSKey                          string
	sqsDeviceRegistrationURL          string
	sqsDeviceDeregistrationURL        string
	sqsNotificationURL                string
	snsAppleDeviceRegistrationTopic   string
	snsAndroidDeviceRegistrationTopic string
	awsAccessKey                      string
	awsSecretKey                      string
	awsRegion                         string
	directoryServiceAddress           string
	settingsServiceAddress            string
	webDomain                         string
	kmsKeyARN                         string
}

func init() {
	flag.StringVar(&config.dbHost, "db_host", "localhost", "the host at which we should attempt to connect to the database")
	flag.IntVar(&config.dbPort, "db_port", 3306, "the port on which we should attempt to connect to the database")
	flag.StringVar(&config.dbName, "db_name", "notification", "the name of the database which we should connect to")
	flag.StringVar(&config.dbUser, "db_user", "baymax-notif", "the name of the user we should connext to the database as")
	flag.StringVar(&config.dbPassword, "db_password", "baymax-notif", "the password we should use when connecting to the database")
	flag.StringVar(&config.dbCACert, "db_ca_cert", "", "the ca cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSCert, "db_tls_cert", "", "the tls cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSKey, "db_tls_key", "", "the tls key to use when connecting to the database")
	flag.StringVar(&config.sqsDeviceRegistrationURL, "sqs_device_registration_url", "", "the sqs url for device registration messages")
	flag.StringVar(&config.sqsDeviceDeregistrationURL, "sqs_device_deregistration_url", "", "the sqs url for device deregistration messages")
	flag.StringVar(&config.sqsNotificationURL, "sqs_notification_url", "", "the sqs url for outgoing notifications")
	flag.StringVar(&config.snsAppleDeviceRegistrationTopic, "sns_apple_device_registration_arn", "", "the arn of the sns topic for apple device push registration")
	flag.StringVar(&config.snsAndroidDeviceRegistrationTopic, "sns_android_device_registration_arn", "", "the arn of the sns topic for android device push registration")
	flag.StringVar(&config.awsAccessKey, "aws_access_key", "", "access key for aws")
	flag.StringVar(&config.awsSecretKey, "aws_secret_key", "", "secret key for aws")
	flag.StringVar(&config.awsRegion, "aws_region", "us-east-1", "aws region")
	flag.StringVar(&config.directoryServiceAddress, "directory_addr", "", "host:port of directory service")
	flag.StringVar(&config.settingsServiceAddress, "settings_addr", "", "host:port of settings service")
	flag.StringVar(&config.webDomain, "web_domain", "", "the baymax web domain")
	flag.StringVar(&config.kmsKeyARN, "kms_key_arn", "", "the arn of the master key that should be used to encrypt outbound and decrypt inbound data")
}

func main() {
	boot.ParseFlags("NOTIFICATION_SERVICE_")
	boot.InitService()

	golog.Infof("Initializing database connection on %s:%d, user: %s, db: %s...", config.dbHost, config.dbPort, config.dbUser, config.dbName)
	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:     config.dbHost,
		Port:     config.dbPort,
		Name:     config.dbName,
		User:     config.dbUser,
		Password: config.dbPassword,
		CACert:   config.dbCACert,
		TLSCert:  config.dbTLSCert,
		TLSKey:   config.dbTLSKey,
	})
	if err != nil {
		golog.Fatalf("failed to initialize db connection: %s", err)
	}

	// generate the SQS and SNS clients we'll need
	baseConfig := &cfg.BaseConfig{
		AppName:      "notification",
		AWSRegion:    config.awsRegion,
		AWSSecretKey: config.awsSecretKey,
		AWSAccessKey: config.awsAccessKey,
	}

	if config.kmsKeyARN == "" {
		golog.Fatalf("KMS key not configured")
	}
	if config.webDomain == "" {
		golog.Fatalf("Web domain not configured")
	}
	if config.directoryServiceAddress == "" {
		golog.Fatalf("Directory service not configured")
	}
	directoryConn, err := grpc.Dial(config.directoryServiceAddress, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	defer directoryConn.Close()
	directoryClient := directory.NewDirectoryClient(directoryConn)

	settingsConn, err := grpc.Dial(config.settingsServiceAddress, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to settings service: %s", err)
	}
	defer settingsConn.Close()
	settingsClient := settings.NewSettingsClient(settingsConn)

	// register the settings with the service
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = settings.RegisterConfigs(
		ctx,
		settingsClient,
		[]*settings.Config{
			nsettings.ReceiveNotificationsConfig,
			nsettings.NotificationPreferenceConfig,
		})
	if err != nil {
		golog.Fatalf("Unable to register configs with the settings service: %s", err.Error())
	}

	eSQS, err := awsutil.NewEncryptedSQS(config.kmsKeyARN, kms.New(baseConfig.AWSSession()), sqs.New(baseConfig.AWSSession()))
	if err != nil {
		golog.Fatalf("Unable to initialize Encrypted SQS: %s", err)
	}

	svc := service.New(
		dal.New(db),
		directoryClient,
		settingsClient,
		&service.Config{
			DeviceRegistrationSQSURL:        config.sqsDeviceRegistrationURL,
			DeviceDeregistrationSQSURL:      config.sqsDeviceDeregistrationURL,
			NotificationSQSURL:              config.sqsNotificationURL,
			AppleDeviceRegistrationSNSARN:   config.snsAppleDeviceRegistrationTopic,
			AndriodDeviceRegistrationSNSARN: config.snsAndroidDeviceRegistrationTopic,
			SQSAPI: eSQS,
			// do not use an encrypted sns client here since these are messages being sent to the client through SNS
			SNSAPI:    sns.New(baseConfig.AWSSession()),
			WebDomain: config.webDomain,
		})
	svc.Start()

	boot.WaitForTermination()
}
