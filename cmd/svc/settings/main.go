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
	cfg "github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/settings"
	"google.golang.org/grpc"
)

var config struct {

	// local dynamodb setup for testing
	localDynamoDBEndpoint           string
	dyanmoDBSettingsTableName       string
	dynamoDBSettingsConfigTableName string

	// aws
	awsAccessKey string
	awsSecretKey string
	awsRegion    string

	// environment
	debug bool
	env   string
	port  int
}

func init() {
	flag.StringVar(&config.localDynamoDBEndpoint, "local_dynamodb_endpoint", "", "local dynamodb endpoint for testing")
	flag.StringVar(&config.dyanmoDBSettingsTableName, "dynamodb_table_name_settings", "", "table name where settings are stored")
	flag.StringVar(&config.dynamoDBSettingsConfigTableName, "dynamodb_table_name_setting_configs", "", "table name where setting configs are stored")
	flag.StringVar(&config.awsRegion, "aws_region", "us-east-1", "AWS region")
	flag.StringVar(&config.awsAccessKey, "aws_access_key", "", "AWS access key")
	flag.StringVar(&config.awsSecretKey, "aws_secret_key", "", "AWS secret key")
	flag.BoolVar(&config.debug, "debug", false, "debug mode")
	flag.StringVar(&config.env, "env", "", "environment")
	flag.IntVar(&config.port, "port", 50053, "port on which to run settings service")
}

func main() {
	boot.ParseFlags("SETTINGS_")

	if config.debug {
		golog.Default().SetLevel(golog.DEBUG)
	}

	baseConfig := &cfg.BaseConfig{
		AppName:      "settings",
		AWSRegion:    config.awsRegion,
		AWSSecretKey: config.awsSecretKey,
		AWSAccessKey: config.awsAccessKey,
		Environment:  config.env,
	}
	awsSession := baseConfig.AWSSession()

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
	server := grpc.NewServer()
	settings.RegisterSettingsServer(server, settingsService)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.port))
	if err != nil {
		golog.Fatalf(err.Error())
	}

	golog.Infof("Starting settings service on port %d", config.port)
	if err := server.Serve(lis); err != nil {
		golog.Fatalf(err.Error())
	}
}
