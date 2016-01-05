package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

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
	debug                bool
	env                  string
}

func init() {
	flag.StringVar(&config.directoryServiceURL, "directory.endpoint", "", "url to talk to the directory service")
	flag.StringVar(&config.threadServiceURL, "threading.endpoint", "", "url to talk to the thread service")
	flag.StringVar(&config.excommsServiceURL, "excomms.endpoint", "", "url to talk to the thread service")
	flag.StringVar(&config.awsAccessKey, "aws.access_key", "", "access key for aws")
	flag.StringVar(&config.awsSecretKey, "aws.secret_key", "", "secret key for aws")
	flag.StringVar(&config.awsRegion, "aws.region", "us-east-1", "aws region")
	flag.StringVar(&config.env, "env", "dev", "environment")
	flag.StringVar(&config.externalMessageQueue, "queue.external_message", "", "queue name for receiving external messages")
	flag.StringVar(&config.inAppMessageQueue, "queue.inapp_message", "", "queue name for receiving in app messages")
	flag.BoolVar(&config.debug, "debug", false, "flag to turn debug on")
}

func main() {
	boot.ParseFlags("ROUTING_")

	if config.debug {
		golog.Default().SetLevel(golog.DEBUG)
	}

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
		Environment:  config.env,
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

	// Wait for an external process interrupt to quit the program
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		golog.Infof("Quitting due to signal %s", sig.String())
		break
	}
}
