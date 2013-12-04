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

	"carefront/config"
	"carefront/libs/svcreg"
	"carefront/services/auth"
	"carefront/thriftapi"
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
	ListenAddr string   `long:"listen" description:"Address:port to listen on"`
	DB         DBConfig `group:"Database" toml:"database"`
}

var DefaultConfig = Config{
	ListenAddr: ":10001",
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
	if conf.Environment == "" || (conf.Environment != "prod" && conf.Environment != "staging" && conf.Environment != "dev") {
		log.Fatal("flag --env is required and must be one of prod, staging, or dev")
	}

	if conf.DB.User == "" || conf.DB.Password == "" || conf.DB.Host == "" || conf.DB.Name == "" {
		fmt.Fprintf(os.Stderr, "Missing either one of user, password, host, or name for the database.\n")
		os.Exit(1)
	}

	metricsRegistry := metrics.NewRegistry().Scope("secure")
	conf.BaseConfig.Stats.StartReporters(metricsRegistry)

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
		DB: db,
	}
	serv := rpc.NewServer()
	if err := serv.RegisterName("Thrift", &thriftapi.AuthServer{Implementation: authService}); err != nil {
		log.Fatal(err)
	}

	service := &config.Server{
		Config:     conf.BaseConfig,
		ListenAddr: conf.ListenAddr,
		ServiceID:  svcreg.ServiceId{Environment: conf.Environment, Name: serviceName},
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
