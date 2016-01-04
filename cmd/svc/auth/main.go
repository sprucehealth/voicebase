package main

import (
	"flag"
	"net"
	"strconv"

	"google.golang.org/grpc"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/auth/internal/server"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	pb "github.com/sprucehealth/backend/svc/auth"
)

var config struct {
	listenPort int64
	debug      bool
	dbHost     string
	dbPort     int64
	dbName     string
	dbUser     string
	dbPassword string
	dbCACert   string
	dbTLSCert  string
	dbTLSKey   string
}

func init() {
	flag.Int64Var(&config.listenPort, "rpc.listen.port", 50051, "the port on which to listen for rpc call")
	flag.BoolVar(&config.debug, "debug", false, "enables golog debug logging for the application")
	flag.StringVar(&config.dbHost, "db.host", "localhost", "the host at which we should attempt to connect to the database")
	flag.Int64Var(&config.dbPort, "db.port", 3306, "the port on which we should attempt to connect to the database")
	flag.StringVar(&config.dbName, "db.name", "auth", "the name of the database which we should connect to")
	flag.StringVar(&config.dbUser, "db.user", "baymax-auth", "the name of the user we should connext to the database as")
	flag.StringVar(&config.dbPassword, "db.password", "baymax-auth", "the password we should use when connecting to the database")
	flag.StringVar(&config.dbCACert, "db.ca.cert", "", "the ca cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSCert, "db.tls.cert", "", "the tls cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSKey, "db.tls.key", "", "the tls key to use when connecting to the database")
}

func main() {
	boot.ParseFlags("AUTH_SERVICE_")
	configureLogging()

	listenAddress := ":" + strconv.FormatInt(config.listenPort, 10)
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
		golog.Fatalf("failed to iniitlize db connection: %s", err)
	}
	s := grpc.NewServer()
	pb.RegisterAuthServer(s, server.New(dal.New(db)))
	golog.Infof("Starting AuthService on %s...", listenAddress)
	s.Serve(lis)
}

func configureLogging() {
	if config.debug {
		golog.Default().SetLevel(golog.DEBUG)
		golog.Debugf("Debug logging enabled...")
	}
}
