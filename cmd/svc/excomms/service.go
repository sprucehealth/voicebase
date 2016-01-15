package main

import (
	"fmt"
	"net"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/server"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/worker"
	cfg "github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"google.golang.org/grpc"
)

func runService() {

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

	baseConfig := &cfg.BaseConfig{
		AppName:      "excomms",
		AWSRegion:    config.awsRegion,
		AWSSecretKey: config.awsSecretKey,
		AWSAccessKey: config.awsAccessKey,
		Environment:  config.env,
	}

	awsSession := baseConfig.AWSSession()
	snsCLI := sns.New(awsSession)

	directoryConn, err := grpc.Dial(
		config.directoryServiceURL,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with directory service: %s", err.Error())
		return
	}
	defer directoryConn.Close()

	store := storage.NewS3(awsSession, config.attachmentBucket, config.attachmentPrefix)

	dl := dal.NewDAL(db)
	w, err := worker.NewWorker(
		awsSession,
		config.incomingRawMessageQueue,
		snsCLI,
		config.externalMessageTopic,
		dl,
		store,
		config.twilioAccountSID,
		config.twilioAuthToken)

	if err != nil {
		golog.Fatalf("Unable to start worker: %s", err.Error())
	}
	w.Start()

	excommsService := server.NewService(
		config.twilioAccountSID,
		config.twilioAuthToken,
		config.twilioApplicationSID,
		dl,
		config.excommsAPIURL,
		directory.NewDirectoryClient(directoryConn),
		snsCLI,
		config.externalMessageTopic,
		clock.New(),
		server.NewSendgridClient(config.sendgridAPIKey),
		server.NewIDGenerator())
	excommsServer := grpc.NewServer()
	excomms.RegisterExCommsServer(excommsServer, excommsService)

	// TODO: Only listen on secure connection.
	golog.Infof("Starting excomms service on port %d", config.excommsServicePort)
	if err := excommsServer.Serve(lis); err != nil {
		golog.Fatalf(err.Error())
	}
}
