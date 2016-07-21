package main

import (
	"flag"
	"net"
	"strconv"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/directory/internal/server"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
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
	dbTLS                string
	dbMaxOpenConnections int
	dbMaxIdleConnections int
}

func init() {
	flag.IntVar(&config.listenPort, "rpc_listen_port", 50051, "the port on which to listen for rpc call")
	flag.StringVar(&config.dbHost, "db_host", "localhost", "the host at which we should attempt to connect to the database")
	flag.IntVar(&config.dbPort, "db_port", 3306, "the port on which we should attempt to connect to the database")
	flag.StringVar(&config.dbName, "db_name", "directory", "the name of the database which we should connect to")
	flag.StringVar(&config.dbUser, "db_user", "baymax-directory", "the name of the user we should connext to the database as")
	flag.StringVar(&config.dbPassword, "db_password", "baymax-directory", "the password we should use when connecting to the database")
	flag.StringVar(&config.dbCACert, "db_ca_cert", "", "the ca cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSCert, "db_tls_cert", "", "the tls cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSKey, "db_tls_key", "", "the tls key to use when connecting to the database")
	flag.StringVar(&config.dbTLS, "db_tls", "false", "Enable TLS for database connection (one of 'true', 'false', 'skip-verify'). Ignored if CA cert provided.")
	flag.IntVar(&config.dbMaxOpenConnections, "db_max_open_connections", 0, "the maximum amount of open connections to have with the database")
	flag.IntVar(&config.dbMaxIdleConnections, "db_max_idle_connections", 0, "the maximum amount of idle connections to have with the database")
}

func main() {
	svc := boot.NewService("directory", nil)

	listenAddress := ":" + strconv.Itoa(config.listenPort)
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		golog.Fatalf("failed to listen: %v", err)
	}
	golog.Infof("Initializing database connection on %s:%d, user: %s, db: %s...", config.dbHost, config.dbPort, config.dbUser, config.dbName)
	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:               config.dbHost,
		Port:               config.dbPort,
		Name:               config.dbName,
		User:               config.dbUser,
		Password:           config.dbPassword,
		CACert:             config.dbCACert,
		TLSCert:            config.dbTLSCert,
		TLSKey:             config.dbTLSKey,
		EnableTLS:          config.dbTLS == "true" || config.dbTLS == "skip-verify",
		SkipVerifyTLS:      config.dbTLS == "skip-verify",
		MaxOpenConnections: config.dbMaxOpenConnections,
		MaxIdleConnections: config.dbMaxIdleConnections,
	})
	if err != nil {
		golog.Fatalf("failed to initialize db connection: %s", err)
	}
	srvMetricsRegistry := svc.MetricsRegistry.Scope("server")
	srv := server.New(dal.New(db), srvMetricsRegistry)
	directory.InitMetrics(srv, srvMetricsRegistry)
	s := svc.NewGRPCServer()
	directory.RegisterDirectoryServer(s, srv)
	golog.Infof("Starting DirectoryService on %s...", listenAddress)
	go s.Serve(lis)

	boot.WaitForTermination()
	lis.Close()
	svc.Shutdown()
}
