package main

import (
	"flag"
	"net"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/auth/internal/server"
	authSetting "github.com/sprucehealth/backend/cmd/svc/auth/internal/settings"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	pb "github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var config struct {
	listenPort                int
	dbHost                    string
	dbPort                    int
	dbName                    string
	dbUser                    string
	dbPassword                string
	dbCACert                  string
	dbTLSCert                 string
	dbTLSKey                  string
	settingsServiceAddress    string
	clientEncryptionKeySecret string
}

func init() {
	flag.IntVar(&config.listenPort, "rpc_listen_port", 50051, "the port on which to listen for rpc call")
	flag.StringVar(&config.dbHost, "db_host", "localhost", "the host at which we should attempt to connect to the database")
	flag.IntVar(&config.dbPort, "db_port", 3306, "the port on which we should attempt to connect to the database")
	flag.StringVar(&config.dbName, "db_name", "auth", "the name of the database which we should connect to")
	flag.StringVar(&config.dbUser, "db_user", "baymax-auth", "the name of the user we should connext to the database as")
	flag.StringVar(&config.dbPassword, "db_password", "baymax-auth", "the password we should use when connecting to the database")
	flag.StringVar(&config.dbCACert, "db_ca_cert", "", "the ca cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSCert, "db_tls_cert", "", "the tls cert to use when connecting to the database")
	flag.StringVar(&config.dbTLSKey, "db_tls_key", "", "the tls key to use when connecting to the database")
	flag.StringVar(&config.settingsServiceAddress, "settings_addr", "", "host:port of settings service")
	flag.StringVar(&config.clientEncryptionKeySecret, "client_encryption_key_secret", "", "the secret to use when generating the disk cache encryption keys for client")
}

func main() {
	metricsRegistry := boot.InitService("auth")

	if config.clientEncryptionKeySecret == "" {
		golog.Fatalf("Client encryption key secret required")
	}
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
		golog.Fatalf("failed to iniitlize db connection: %s", err)
	}

	settingsConn, err := grpc.Dial(config.settingsServiceAddress, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to settings service: %s", err)
	}
	defer settingsConn.Close()
	settingsClient := settings.NewSettingsClient(settingsConn)

	// register the settings with the service
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = settings.RegisterConfigs(
		ctx,
		settingsClient,
		[]*settings.Config{
			authSetting.Enable2FAConfig,
		})
	if err != nil {
		golog.Fatalf("Unable to register configs with the settings service: %s", err.Error())
	}
	aSrv, err := server.New(dal.New(db), settingsClient, config.clientEncryptionKeySecret)
	if err != nil {
		golog.Fatalf("Error while initializing auth server: %s", err)
	}
	pb.InitMetrics(aSrv, metricsRegistry)

	s := grpc.NewServer()
	pb.RegisterAuthServer(s, aSrv)
	golog.Infof("Starting AuthService on %s...", listenAddress)
	go s.Serve(lis)

	boot.WaitForTermination()
}
