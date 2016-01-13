package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
)

var config struct {
	excommsServicePort      int
	excommsAPIURL           string
	directoryServiceURL     string
	twilioAuthToken         string
	twilioAccountSID        string
	twilioApplicationSID    string
	sendgridAPIKey          string
	awsRegion               string
	awsAccessKey            string
	awsSecretKey            string
	externalMessageTopic    string
	incomingRawMessageQueue string
	debug                   bool
	dbHost                  string
	dbPassword              string
	dbName                  string
	dbUserName              string
	dbPort                  int
	httpAddr                string
	proxyProtocol           bool
	excommsServiceURL       string
	incomingRawMessageTopic string
	env                     string
}

func init() {
	flag.IntVar(&config.excommsServicePort, "excomms.port", 5200, "port on which excomms service should listen")
	flag.StringVar(&config.excommsAPIURL, "excommsapi.endpoint", "", "url for excomms api")
	flag.StringVar(&config.twilioAccountSID, "twilio.account_sid", "", "account sid for twilio account")
	flag.StringVar(&config.twilioApplicationSID, "twilio.application_sid", "", "application sid for twilio")
	flag.StringVar(&config.twilioAuthToken, "twilio.auth_token", "", "auth token for twilio account")
	flag.StringVar(&config.directoryServiceURL, "directory.endpoint", "", "url to connect with directory service")
	flag.StringVar(&config.awsRegion, "aws.region", "us-east-1", "aws region")
	flag.StringVar(&config.awsAccessKey, "aws.access_key", "", "access key for aws user")
	flag.StringVar(&config.awsSecretKey, "aws.secret_key", "", "secret key for aws user")
	flag.StringVar(&config.sendgridAPIKey, "sendgrid.api_key", "", "sendgrid api key")
	flag.StringVar(&config.externalMessageTopic, "sns.external_message_topic", "", "sns topic on which to post external message events")
	flag.BoolVar(&config.debug, "debug", false, "debug flag")
	flag.StringVar(&config.dbHost, "db.host", "", "database host")
	flag.StringVar(&config.dbPassword, "db.password", "", "database password")
	flag.StringVar(&config.dbName, "db.name", "", "database name")
	flag.StringVar(&config.dbUserName, "db.username", "", "database username")
	flag.IntVar(&config.dbPort, "db.port", 3306, "database port")
	flag.StringVar(&config.incomingRawMessageQueue, "queue.incoming_raw_message", "", "queue name for receiving incoming raw messages")
	flag.StringVar(&config.httpAddr, "http", "0.0.0.0:8900", "listen for http on `host:port`")
	flag.BoolVar(&config.proxyProtocol, "proxyproto", false, "enabled proxy protocol")
	flag.StringVar(&config.excommsServiceURL, "excomms.url", "localhost:5200", "url for events processor service. format `host:port`")
	flag.StringVar(&config.incomingRawMessageTopic, "sns.incoming_raw_message_topic", "", "Inbound msg topic")
	flag.StringVar(&config.env, "env", "dev", "environment")
}

func main() {
	boot.ParseFlags("EXCOMMS_")

	conc.Go(func() {
		runAPI()
	})

	conc.Go(func() {
		runService()
	})

	// Wait for an external process interrupt to quit the program
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		golog.Infof("Quitting due to signal %s", sig.String())
		break
	}

}
