package main

import (
	"flag"
	"net"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/server"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/payments"
)

var (
	flagRPCListenAddr = flag.String("rpc_listen_addr", "127.0.0.1:50062", "host:port to listen on for rpc requests")
	flagBehindProxy   = flag.Bool("behind_proxy", false, "Flag to indicate when the service is behind a proxy")
	// DB
	flagDBHost     = flag.String("db_host", "localhost", "the host at which we should attempt to connect to the database")
	flagDBPort     = flag.Int("db_port", 3306, "the port on which we should attempt to connect to the database")
	flagDBName     = flag.String("db_name", "payments", "the name of the database which we should connect to")
	flagDBUser     = flag.String("db_user", "baymax-payments", "the name of the user we should connext to the database as")
	flagDBPassword = flag.String("db_password", "baymax-payments", "the password we should use when connecting to the database")
	flagDBCACert   = flag.String("db_ca_cert", "", "the ca cert to use when connecting to the database")
	flagDBTLSCert  = flag.String("db_tls_cert", "", "the tls cert to use when connecting to the database")
	flagDBTLSKey   = flag.String("db_tls_key", "", "the tls key to use when connecting to the database")
	flagDBTLS      = flag.String("db_tls", "skip-verify", "Enable TLS for database connection (one of 'true', 'false', 'skip-verify'). Ignored if CA cert provided.")
)

func main() {
	svc := boot.NewService("payments", nil)

	lis, err := net.Listen("tcp", *flagRPCListenAddr)
	if err != nil {
		golog.Fatalf("failed to listen: %v", err)
	}
	golog.Infof("Initializing database connection on %s:%d, user: %s, db: %s...", *flagDBHost, *flagDBPort, *flagDBUser, *flagDBName)
	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:          *flagDBHost,
		Port:          *flagDBPort,
		Name:          *flagDBName,
		User:          *flagDBUser,
		Password:      *flagDBPassword,
		CACert:        *flagDBCACert,
		TLSCert:       *flagDBTLSCert,
		TLSKey:        *flagDBTLSKey,
		EnableTLS:     *flagDBTLS == "true" || *flagDBTLS == "skip-verify",
		SkipVerifyTLS: *flagDBTLS == "skip-verify",
	})
	if err != nil {
		golog.Fatalf("failed to initialize db connection: %s", err)
	}

	pSrv, err := server.New(dal.New(db))
	if err != nil {
		golog.Fatalf("Error while initializing payments server: %s", err)
	}
	payments.InitMetrics(pSrv, svc.MetricsRegistry.Scope("server"))

	s := svc.GRPCServer()
	payments.RegisterPaymentsServer(s, pSrv)
	golog.Infof("Starting PaymentsService on %s...", *flagRPCListenAddr)
	go s.Serve(lis)

	boot.WaitForTermination()
	svc.Shutdown()
}
