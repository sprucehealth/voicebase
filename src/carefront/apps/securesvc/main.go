package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"time"

	"carefront/common/config"
	"carefront/libs/svcreg"
	"carefront/services/auth"
	"carefront/thrift/api"

	_ "github.com/go-sql-driver/mysql"
	"github.com/samuel/go-metrics/metrics"
	"github.com/samuel/go-thrift/thrift"
)

type DBConfig struct {
	User     string `long:"db_user" description:"Username for accessing database"`
	Password string `long:"db_password" description:"Password for accessing database"`
	Host     string `long:"db_host" description:"Database host"`
	Name     string `long:"db_name" description:"Database name"`
}

type Config struct {
	*config.BaseConfig
	ListenAddr          string   `long:"listen" description:"Address:port to listen on"`
	DB                  DBConfig `group:"Database" toml:"database"`
	AuthTokenExpiration int      `long:"auth_token_expire" description:"Expiration time in seconds for the auth token"`
	AuthTokenRenew      int      `long:"auth_token_renew" description:"Time left below which to renew the auth token"`
}

var DefaultConfig = Config{
	BaseConfig: &config.BaseConfig{
		AppName: "secure",
	},
	ListenAddr:          ":10001",
	AuthTokenExpiration: 60 * 60 * 24 * 2,
	AuthTokenRenew:      60 * 60 * 36,
}

const (
	serviceName = "secure"
)

func main() {
	conf := DefaultConfig
	_, err := config.Parse(&conf)
	if err != nil {
		log.Fatal(err)
	}

	if conf.DB.User == "" || conf.DB.Password == "" || conf.DB.Host == "" || conf.DB.Name == "" {
		fmt.Fprintf(os.Stderr, "Missing either one of user, password, host, or name for the database.\n")
		os.Exit(1)
	}

	metricsRegistry := metrics.NewRegistry()
	conf.StartReporters(metricsRegistry)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", conf.DB.User, conf.DB.Password, conf.DB.Host, conf.DB.Name)

	// this gives us a connection pool to the sql instance
	// without executing any statements against the sql database
	// or checking the network connection and authentication to the database
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	authService := &auth.AuthService{
		DB:             db,
		ExpireDuration: time.Duration(conf.AuthTokenExpiration) * time.Second,
		RenewDuration:  time.Duration(conf.AuthTokenRenew) * time.Second,
	}
	serv := rpc.NewServer()
	if err := serv.RegisterName("Thrift", &api.AuthServer{Implementation: authService}); err != nil {
		log.Fatal(err)
	}

	service := &config.Server{
		Config:          conf.BaseConfig,
		ListenAddr:      conf.ListenAddr,
		MetricsRegistry: metricsRegistry.Scope("securesvc-server"),
		ServiceID:       svcreg.ServiceId{Environment: conf.Environment, Name: serviceName},
		ServFunc: func(conn net.Conn) {
			serv.ServeCodec(thrift.NewServerCodec(thrift.NewFramedReadWriteCloser(conn, 0), thrift.NewBinaryProtocol(true, false)))
		},
	}
	if err := service.Start(); err != nil {
		log.Fatal(err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	_ = <-signalChan
	service.Stop(time.Second * 5)
	time.Sleep(time.Millisecond * 200)
}
