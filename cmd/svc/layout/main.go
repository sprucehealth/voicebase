package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/layout/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/layout/internal/server"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/layout"
	"google.golang.org/grpc"
)

var config struct {
	dbHost        string
	dbPort        int
	dbPassword    string
	dbTLS         string
	dbUserName    string
	dbName        string
	dbCACert      string
	listeningPort int
	s3Bucket      string
	s3Prefix      string
}

func init() {
	flag.StringVar(&config.dbHost, "db_host", "", "database host")
	flag.StringVar(&config.dbPassword, "db_password", "", "database password")
	flag.StringVar(&config.dbName, "db_name", "", "database name")
	flag.StringVar(&config.dbUserName, "db_username", "", "database username")
	flag.IntVar(&config.dbPort, "db_port", 3306, "database port")
	flag.StringVar(&config.dbCACert, "db_ca_cert", "", "Path to database CA certificate")
	flag.StringVar(&config.dbTLS, "db_tls", "", "Enable TLS for database connection (one of 'true', 'false', 'skip-verify'). Ignored if CA cert provided.")
	flag.IntVar(&config.listeningPort, "listening_port", 0, "Port on which SAML service should listen")
	flag.StringVar(&config.s3Bucket, "s3_bucket", "", "name of S3 bucket")
	flag.StringVar(&config.s3Prefix, "s3_prefix", "", "prefix for items in s3 bucket")
}

func main() {
	svc := boot.NewService("layout", nil)

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

	if config.s3Bucket == "" {
		golog.Fatalf("s3_bucket required")
	} else if config.s3Prefix == "" {
		golog.Fatalf("s3_prefix required")
	} else if config.listeningPort == 0 {
		golog.Fatalf("listening_port required")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.listeningPort))
	if err != nil {
		golog.Fatalf(err.Error())
	}

	awsSession, err := svc.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	layoutServer := grpc.NewServer()
	layoutService := server.New(dal.NewDAL(db), layout.NewStore(storage.NewS3(awsSession, config.s3Bucket, config.s3Prefix)))

	layout.InitMetrics(layoutService, svc.MetricsRegistry.Scope("server"))
	layout.RegisterLayoutServer(layoutServer, layoutService)

	conc.Go(func() {
		golog.Infof("Starting layout service on port %d", config.listeningPort)
		if err := layoutServer.Serve(lis); err != nil {
			golog.Fatalf(err.Error())
		}
	})

	boot.WaitForTermination()
}
