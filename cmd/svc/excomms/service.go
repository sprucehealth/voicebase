package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/cleaner"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/server"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/worker"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/twilio"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func runService(bootSvc *boot.Service) {
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

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.excommsServicePort))
	if err != nil {
		golog.Fatalf(err.Error())
	}

	awsSession, err := bootSvc.AWSSession()
	if err != nil {
		golog.Fatalf("Failed to create AWS session: %s", err)
	}

	eSNS, err := awsutil.NewEncryptedSNS(config.kmsKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
		return
	}
	eSQS, err := awsutil.NewEncryptedSQS(config.kmsKeyARN, kms.New(awsSession), sqs.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sqs: %s", err.Error())
		return
	}

	directoryConn, err := grpc.Dial(
		config.directoryServiceURL,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with directory service: %s", err.Error())
		return
	}
	defer directoryConn.Close()

	settingsConn, err := grpc.Dial(
		config.settingsServiceURL,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with settings service: %s", err.Error())
		return
	}
	defer settingsConn.Close()

	if config.notificationSQSURL == "" {
		golog.Fatalf("notification_sqs_url flag required")
	}
	notificationClient := notification.NewClient(eSQS, &notification.ClientConfig{SQSNotificationURL: config.notificationSQSURL})

	// register the settings with the service
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	settingsClient := settings.NewSettingsClient(settingsConn)
	_, err = settings.RegisterConfigs(
		ctx,
		settingsClient,
		[]*settings.Config{
			excommsSettings.NumbersToRingConfig,
			excommsSettings.VoicemailOptionConfig,
			excommsSettings.SendCallsToVoicemailConfig,
			excommsSettings.TranscribeVoicemailConfig,
			excommsSettings.AfterHoursGreetingOptionConfig,
			excommsSettings.AfterHoursVoicemailEnabledConfig,
			excommsSettings.ForwardingListTimeoutConfig,
			excommsSettings.PauseBeforeCallConnectConfig,
		})
	if err != nil {
		golog.Fatalf("Unable to register configs with the settings service: %s", err.Error())
	}

	store := storage.NewS3(awsSession, config.attachmentBucket, config.attachmentPrefix)
	dl := dal.New(db, clock.New())
	w, err := worker.NewWorker(
		config.incomingRawMessageQueue,
		eSNS,
		eSQS,
		config.externalMessageTopic,
		dl,
		store,
		config.twilioAccountSID,
		config.twilioAuthToken,
		config.resourceCleanerTopic)

	if err != nil {
		golog.Fatalf("Unable to start worker: %s", err.Error())
	}
	w.Start()

	proxyNumberManager := proxynumber.NewManager(dl, clock.New())

	if config.apiDomain == "" {
		golog.Fatalf("api_domain is required")
	}
	if config.sigKeys == "" {
		golog.Fatalf("signature_keys_csv is required")
	}

	sigKeys := strings.Split(config.sigKeys, ",")
	sigKeysByteSlice := make([][]byte, len(sigKeys))
	for i, sk := range sigKeys {
		sigKeysByteSlice[i] = []byte(sk)
	}
	signer, err := sig.NewSigner(sigKeysByteSlice, nil)
	if err != nil {
		golog.Fatalf("Failed to create signer: %s", err.Error())
	}
	ms := urlutil.NewSigner("https://"+config.mediaAPIDomain, signer, clock.New())

	excommsService := server.NewService(
		config.twilioAccountSID,
		config.twilioAuthToken,
		config.twilioApplicationSID,
		config.twilioSigningKeySID,
		config.twilioSigningKey,
		config.twilioVideoConfigSID,
		dl,
		config.excommsAPIURL,
		directory.NewDirectoryClient(directoryConn),
		eSNS,
		config.externalMessageTopic,
		config.eventTopic,
		clock.New(),
		server.NewSendgridClient(config.sendgridAPIKey),
		server.NewIDGenerator(),
		proxyNumberManager,
		ms,
		notificationClient)
	excomms.InitMetrics(excommsService, bootSvc.MetricsRegistry.Scope("server"))

	excommsServer := grpc.NewServer()
	excomms.RegisterExCommsServer(excommsServer, excommsService)

	res, err := eSQS.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: ptr.String(config.resourceCleanerQueueURL),
	})
	if err != nil {
		golog.Fatalf("Unable to build queue url for resource cleaner %s: %s", config.resourceCleanerQueueURL, err.Error())
	}

	resourceCleaner := cleaner.NewWorker(twilio.NewClient(config.twilioAccountSID, config.twilioAuthToken, nil), dl, eSQS, *res.QueueUrl)
	resourceCleaner.Start()
	// TODO: Only listen on secure connection.
	golog.Infof("Starting excomms service on port %d", config.excommsServicePort)
	if err := excommsServer.Serve(lis); err != nil {
		golog.Fatalf(err.Error())
	}
}
