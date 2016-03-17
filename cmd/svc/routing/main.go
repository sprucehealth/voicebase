package main

import (
	"flag"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/routing/internal"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

var config struct {
	directoryServiceURL   string
	threadServiceURL      string
	excommsServiceURL     string
	externalMessageQueue  string
	inAppMessageQueue     string
	kmsKeyARN             string
	blockAccountsTopicARN string
}

func init() {
	flag.StringVar(&config.directoryServiceURL, "directory_addr", "", "`host:port` to connect to directory service")
	flag.StringVar(&config.threadServiceURL, "threading_addr", "", "`host:port`to connect to threading service")
	flag.StringVar(&config.excommsServiceURL, "excomms_addr", "", "`host:port` to connect to excomms service")
	flag.StringVar(&config.externalMessageQueue, "queue_external_message", "", "queue name for receiving external messages")
	flag.StringVar(&config.inAppMessageQueue, "queue_inapp_message", "", "queue name for receiving in app messages")
	flag.StringVar(&config.kmsKeyARN, "kms_key_arn", "", "the arn of the master key that should be used to encrypt outbound and decrypt inbound data")
	flag.StringVar(&config.blockAccountsTopicARN, "block_accounts_topic_arn", "", "arn of the block accounts sns topic")
}

func main() {
	boot.InitService("routing")

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

	awsSession, err := boot.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	eSNS, err := awsutil.NewEncryptedSNS(config.kmsKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
		return
	}

	routingService, err := internal.NewRoutingService(
		awsSession,
		config.externalMessageQueue,
		config.inAppMessageQueue,
		directory.NewDirectoryClient(directoryConn),
		threading.NewThreadsClient(threadConn),
		excomms.NewExCommsClient(excommsConn),
		eSNS,
		config.blockAccountsTopicARN,
		config.kmsKeyARN,
	)
	if err != nil {
		golog.Fatalf(err.Error())
		return
	}

	golog.Infof("Started routing service ...")
	routingService.Start()

	boot.WaitForTermination()
}
