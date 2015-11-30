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
	pb "github.com/sprucehealth/backend/svc/directory"
	"google.golang.org/grpc"
)

var config struct {
	listenPort           int
	debug                bool
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
}

func init() {
	flag.IntVar(&config.listenPort, "rpc.listen.port", 50051, "the port on which to listen for rpc call")
	flag.BoolVar(&config.debug, "debug", false, "enables golog debug logging for the application")
	flag.StringVar(&config.dbHost, "db.host", "localhost", "the host at which we should attempt to connect to the database")
	flag.IntVar(&config.dbPort, "db.port", 3306, "the port on which we should attempt to connect to the database")
	flag.StringVar(&config.dbName, "db.name", "directory", "the name of the database which we should connect to")
	flag.StringVar(&config.dbUser, "db.user", "baymax-directory", "the name of the user we should connext to the database as")
	flag.StringVar(&config.dbPassword, "db.password", "baymax-directory", "the password we should use when connecting to the database")
	flag.StringVar(&config.dbCACert, "db.ca.cert", "", "the ca cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSCert, "db.tls.cert", "", "the tls cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSKey, "db.tls.key", "", "the tls key to use when connecting to the database")
	flag.IntVar(&config.dbMaxOpenConnections, "db.max.open.connections", 0, "the maximum amount of open connections to have with the database")
	flag.IntVar(&config.dbMaxIdleConnections, "db.max.idle.connections", 0, "the maximum amount of idle connections to have with the database")
}

func main() {
	boot.ParseFlags("AUTH_SERVICE_")
	configureLogging()

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
		MaxOpenConnections: config.dbMaxOpenConnections,
		MaxIdleConnections: config.dbMaxIdleConnections,
	})
	if err != nil {
		golog.Fatalf("failed to iniitlize db connection: %s", err)
	}
	s := grpc.NewServer()
	pb.RegisterDirectoryServer(s, server.New(dal.New(db)))
	golog.Infof("Starting DirectoryService on %s...", listenAddress)
	s.Serve(lis)
}

func configureLogging() {
	if config.debug {
		golog.Default().SetLevel(golog.DEBUG)
		golog.Debugf("Debug logging enabled...")
	}
}
