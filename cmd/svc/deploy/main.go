package main

import (
	"flag"
	"net"
	"strconv"

	"google.golang.org/grpc"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/deploy/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/deploy/internal/deployment"
	"github.com/sprucehealth/backend/cmd/svc/deploy/internal/server"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/deploy"
)

var config struct {
	listenPort           int
	dbHost               string
	dbPort               int
	dbName               string
	dbUser               string
	dbPassword           string
	dbCACert             string
	dbTLSCert            string
	dbTLSKey             string
	dbMaxOpenConnections int
	dbMaxIdleConnections int
	eventsQueueURL       string
}

func init() {
	flag.IntVar(&config.listenPort, "rpc_listen_port", 51050, "the port on which to listen for rpc call")
	flag.StringVar(&config.dbHost, "db_host", "localhost", "the host at which we should attempt to connect to the database")
	flag.IntVar(&config.dbPort, "db_port", 3306, "the port on which we should attempt to connect to the database")
	flag.StringVar(&config.dbName, "db_name", "deploy", "the name of the database which we should connect to")
	flag.StringVar(&config.dbUser, "db_user", "deploy", "the name of the user we should connext to the database as")
	flag.StringVar(&config.dbPassword, "db_password", "deploy", "the password we should use when connecting to the database")
	flag.StringVar(&config.dbCACert, "db_ca_cert", "", "the ca cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSCert, "db_tls_cert", "", "the tls cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSKey, "db_tls_key", "", "the tls key to use when connecting to the database")
	flag.StringVar(&config.eventsQueueURL, "sqs_events_url", "", "the url of the sqs queue containing build events")
}

func main() {
	svc := boot.NewService("deploy")
	validateArgs()
	listenAddress := ":" + strconv.Itoa(config.listenPort)
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		golog.Fatalf("failed to listen: %v", err)
	}
	golog.Infof("Initializing database connection on %s:%d, user: %s, db: %s...", config.dbHost, config.dbPort, config.dbUser, config.dbName)
	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:     config.dbHost,
		Port:     config.dbPort,
		Name:     config.dbName,
		User:     config.dbUser,
		Password: config.dbPassword,
		CACert:   config.dbCACert,
		TLSCert:  config.dbTLSCert,
		TLSKey:   config.dbTLSKey,
	})
	if err != nil {
		golog.Fatalf("failed to initialize db connection: %s", err)
	}
	dl := dal.New(db)
	awsSession, err := svc.AWSSession()
	if err != nil {
		golog.Fatalf("Error while getting AWS Session: %s", err)
	}
	dMan := deployment.NewManager(dl, awsSession, config.eventsQueueURL)
	dMan.Start()
	defer dMan.Stop()

	srvMetricsRegistry := svc.MetricsRegistry.Scope("server")
	srv := server.New(dl, dMan)
	deploy.InitMetrics(srv, srvMetricsRegistry)
	s := grpc.NewServer()
	deploy.RegisterDeployServer(s, srv)
	golog.Infof("Starting DeployService on %s...", listenAddress)
	go s.Serve(lis)

	boot.WaitForTermination()
}

func validateArgs() {
	if config.eventsQueueURL == "" {
		golog.Fatalf("sqs_events_url is required")
	}
}
