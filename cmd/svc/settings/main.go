package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/settings/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/settings/internal/server"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/settings"
)

var config struct {
	// local dynamodb setup for testing
	localDynamoDBEndpoint           string
	dyanmoDBSettingsTableName       string
	dynamoDBSettingsConfigTableName string

	// environment
	port int
}

func init() {
	flag.StringVar(&config.localDynamoDBEndpoint, "local_dynamodb_endpoint", "", "local dynamodb endpoint for testing")
	flag.StringVar(&config.dyanmoDBSettingsTableName, "dynamodb_table_name_settings", "", "table name where settings are stored")
	flag.StringVar(&config.dynamoDBSettingsConfigTableName, "dynamodb_table_name_setting_configs", "", "table name where setting configs are stored")
	flag.IntVar(&config.port, "port", 50053, "port on which to run settings service")
}

func main() {
	bootSvc := boot.NewService("settings", nil)

	awsSession, err := bootSvc.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	dynamoDBClient := dynamodb.New(func() *session.Session {
		if config.localDynamoDBEndpoint != "" {
			golog.Infof("AWS Dynamo DB Endpoint configured as %s...", config.localDynamoDBEndpoint)
			dynamoConfig := &aws.Config{
				Region:     ptr.String("us-east-1"),
				DisableSSL: ptr.Bool(true),
				Endpoint:   &config.localDynamoDBEndpoint,
			}
			return session.New(dynamoConfig)
		}
		return awsSession
	}())

	dal := dal.New(dynamoDBClient, config.dyanmoDBSettingsTableName, config.dynamoDBSettingsConfigTableName)
	settingsService := server.New(dal)
	settings.InitMetrics(settingsService, bootSvc.MetricsRegistry.Scope("server"))
	server := bootSvc.NewGRPCServer()
	settings.RegisterSettingsServer(server, settingsService)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.port))
	if err != nil {
		golog.Fatalf(err.Error())
	}
	defer lis.Close()

	golog.Infof("Starting settings service on port %d", config.port)
	go func() {
		if err := server.Serve(lis); err != nil {
			golog.Errorf(err.Error())
		}
	}()

	boot.WaitForTermination()
	lis.Close()
	bootSvc.Shutdown()
}
