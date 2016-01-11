package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/notification/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/notification/internal/service"
	cfg "github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
)

var config struct {
	debug                             bool
	dbHost                            string
	dbPort                            int
	dbName                            string
	dbUser                            string
	dbPassword                        string
	dbCACert                          string
	dbTLSCert                         string
	dbTLSKey                          string
	sqsDeviceRegistrationURL          string
	sqsNotificationURL                string
	snsAppleDeviceRegistrationTopic   string
	snsAndroidDeviceRegistrationTopic string
	awsAccessKey                      string
	awsSecretKey                      string
	awsRegion                         string
}

func init() {
	flag.BoolVar(&config.debug, "debug", false, "enables golog debug logging for the application")
	flag.StringVar(&config.dbHost, "db.host", "localhost", "the host at which we should attempt to connect to the database")
	flag.IntVar(&config.dbPort, "db.port", 3306, "the port on which we should attempt to connect to the database")
	flag.StringVar(&config.dbName, "db.name", "notification", "the name of the database which we should connect to")
	flag.StringVar(&config.dbUser, "db.user", "baymax-notif", "the name of the user we should connext to the database as")
	flag.StringVar(&config.dbPassword, "db.password", "baymax-notif", "the password we should use when connecting to the database")
	flag.StringVar(&config.dbCACert, "db.ca.cert", "", "the ca cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSCert, "db.tls.cert", "", "the tls cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSKey, "db.tls.key", "", "the tls key to use when connecting to the database")
	flag.StringVar(&config.sqsDeviceRegistrationURL, "sqs.device.registration.url", "", "the sqs url for device registration messages")
	flag.StringVar(&config.sqsNotificationURL, "sqs.notification.url", "", "the sqs url for outgoing notifications")
	flag.StringVar(&config.snsAppleDeviceRegistrationTopic, "sns.apple.device.registration.arn", "", "the arn of the sns topic for apple device push registration")
	flag.StringVar(&config.snsAndroidDeviceRegistrationTopic, "sns.android.device.registration.arn", "", "the arn of the sns topic for android device push registration")
	flag.StringVar(&config.awsAccessKey, "aws.access.key", "", "access key for aws")
	flag.StringVar(&config.awsSecretKey, "aws.secret.key", "", "secret key for aws")
	flag.StringVar(&config.awsRegion, "aws.region", "us-east-1", "aws region")
}

func main() {
	boot.ParseFlags("NOTIFICATION_SERVICE_")
	configureLogging()

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
		golog.Fatalf("failed to iniitlize db connection: %s", err)
	}

	// generate the SQS and SNS clients we'll need
	baseConfig := &cfg.BaseConfig{
		AppName:      "notification",
		AWSRegion:    config.awsRegion,
		AWSSecretKey: config.awsSecretKey,
		AWSAccessKey: config.awsAccessKey,
	}
	svc := service.New(
		dal.New(db),
		&service.Config{
			Session:                         baseConfig.AWSSession(),
			DeviceRegistrationSQSURL:        config.sqsDeviceRegistrationURL,
			NotificationSQSURL:              config.sqsNotificationURL,
			AppleDeviceRegistrationSNSARN:   config.snsAppleDeviceRegistrationTopic,
			AndriodDeviceRegistrationSNSARN: config.snsAndroidDeviceRegistrationTopic,
		})
	svc.Start()

	// Wait for an external process interrupt to quit the program
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		golog.Infof("Quitting due to signal %s", sig.String())
		svc.Shutdown()
		break
	}
}

func configureLogging() {
	if config.debug {
		golog.Default().SetLevel(golog.DEBUG)
		golog.Debugf("Debug logging enabled...")
	}
}
