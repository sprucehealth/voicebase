package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	cfg "github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"google.golang.org/grpc"
)

var config struct {
	excommsServicePort   int
	excommsAPIURL        string
	directoryServiceURL  string
	twilioAuthToken      string
	twilioAccountSID     string
	twilioApplicationSID string
	awsAccessKey         string
	awsSecretKey         string
	externalMessageTopic string
	debug                bool
	dbHost               string
	dbPassword           string
	dbName               string
	dbUserName           string
	dbPort               int
}

func init() {
	flag.IntVar(&config.excommsServicePort, "excomms.port", 5200, "port on which excomms service should listen")
	flag.StringVar(&config.excommsAPIURL, "excommsapi.endpoint", "", "url for excomms api")
	flag.StringVar(&config.twilioAccountSID, "twilio.account_sid", "", "account sid for twilio account")
	flag.StringVar(&config.twilioApplicationSID, "twilio.application_sid", "", "application sid for twilio")
	flag.StringVar(&config.twilioAuthToken, "twilio.auth_token", "", "auth token for twilio account")
	flag.StringVar(&config.directoryServiceURL, "directory.endpoint", "", "url to connect with directory service")
	flag.StringVar(&config.awsAccessKey, "aws.access_key", "", "access key for aws user")
	flag.StringVar(&config.awsSecretKey, "aws.secret_key", "", "secret key for aws user")
	flag.StringVar(&config.externalMessageTopic, "sns.external_message_topic", "", "sns topic on which to post external message events")
	flag.BoolVar(&config.debug, "debug", false, "debug flag")
	flag.StringVar(&config.dbHost, "db.host", "", "database host")
	flag.StringVar(&config.dbPassword, "db.password", "", "database password")
	flag.StringVar(&config.dbName, "db.name", "", "database name")
	flag.StringVar(&config.dbUserName, "db.username", "", "database username")
	flag.IntVar(&config.dbPort, "db.port", 3306, "database port")
}

func main() {
	boot.ParseFlags("EXCOMMSAPI_")

	dbConfig := &cfg.DB{
		User:     config.dbUserName,
		Password: config.dbPassword,
		Host:     config.dbHost,
		Port:     config.dbPort,
		Name:     config.dbName,
	}

	db, err := dbConfig.ConnectMySQL(nil)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.excommsServicePort))
	if err != nil {
		golog.Fatalf(err.Error())
	}

	if config.debug {
		golog.Default().SetLevel(golog.DEBUG)
	}

	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(config.awsAccessKey, config.awsSecretKey, ""),
		Region:      ptr.String("us-east-1"),
	}
	awsSession := session.New(awsConfig)
	snsCLI := sns.New(awsSession)

	directoryConn, err := grpc.Dial(
		config.directoryServiceURL,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with directory service: %s", err.Error())
		return
	}
	defer directoryConn.Close()

	excommsService := internal.NewService(
		config.twilioAccountSID,
		config.twilioAuthToken,
		config.twilioApplicationSID,
		dal.NewDAL(db),
		config.excommsAPIURL,
		directory.NewDirectoryClient(directoryConn),
		snsCLI,
		config.externalMessageTopic)
	excommsServer := grpc.NewServer()
	excomms.RegisterExCommsServer(excommsServer, excommsService)

	// TODO: Only listen on secure connection.
	golog.Infof("Starting excomms service on port %d", config.excommsServicePort)
	if err := excommsServer.Serve(lis); err != nil {
		golog.Fatalf(err.Error())
	}
}
