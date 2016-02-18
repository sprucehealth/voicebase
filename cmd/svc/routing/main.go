package main

import (
	"flag"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/routing/internal"
	cfg "github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

var config struct {
	directoryServiceURL  string
	threadServiceURL     string
	excommsServiceURL    string
	awsAccessKey         string
	awsSecretKey         string
	awsRegion            string
	externalMessageQueue string
	inAppMessageQueue    string
}

func init() {
	flag.StringVar(&config.directoryServiceURL, "directory_endpoint", "", "url to talk to the directory service")
	flag.StringVar(&config.threadServiceURL, "threading_endpoint", "", "url to talk to the thread service")
	flag.StringVar(&config.excommsServiceURL, "excomms_endpoint", "", "url to talk to the thread service")
	flag.StringVar(&config.awsAccessKey, "aws_access_key", "", "access key for aws")
	flag.StringVar(&config.awsSecretKey, "aws_secret_key", "", "secret key for aws")
	flag.StringVar(&config.awsRegion, "aws_region", "us-east-1", "aws region")
	flag.StringVar(&config.externalMessageQueue, "queue_external_message", "", "queue name for receiving external messages")
	flag.StringVar(&config.inAppMessageQueue, "queue_inapp_message", "", "queue name for receiving in app messages")
}

func main() {
	boot.ParseFlags("ROUTING_")
	boot.InitService()

	directoryConn, err := grpc.Dial(
		config.directoryServiceURL,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with directory service: %s", err.Error())
		return
	}
	defer directoryConn.Close()

	threadConn, err := grpc.Dial(
		config.threadServiceURL,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with thread service: %s", err.Error())
		return
	}
	defer threadConn.Close()

	excommsConn, err := grpc.Dial(
		config.excommsServiceURL,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with excomms service: %s", err.Error())
		return
	}
	defer excommsConn.Close()

	baseConfig := &cfg.BaseConfig{
		AppName:      "routing",
		AWSRegion:    config.awsRegion,
		AWSSecretKey: config.awsSecretKey,
		AWSAccessKey: config.awsAccessKey,
	}

	awsSession := baseConfig.AWSSession()
	routingService, err := internal.NewRoutingService(
		awsSession,
		config.externalMessageQueue,
		config.inAppMessageQueue,
		directory.NewDirectoryClient(directoryConn),
		threading.NewThreadsClient(threadConn),
		excomms.NewExCommsClient(excommsConn),
	)
	if err != nil {
		golog.Fatalf(err.Error())
		return
	}

	golog.Infof("Started routing service ...")
	routingService.Start()

	boot.WaitForTermination()
}
