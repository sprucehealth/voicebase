package main

import (
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"time"

	"carefront/config"
	"carefront/libs/svcreg"
	"carefront/thriftauth"
	"github.com/samuel/go-thrift/thrift"
)

type Config struct {
	*config.BaseConfig
	ListenAddr string `long:"listen" description:"Address:port to listen on"`
}

var DefaultConfig = Config{
	ListenAddr: ":10001",
}

const (
	serviceName = "secure"
)

type authServiceImplementation struct {
}

func (srv *authServiceImplementation) Login(login string, password string) (*thriftauth.AuthResponse, error) {
	println("Login", login, password)
	return &thriftauth.AuthResponse{
		Token:     "token",
		AccountId: 123,
	}, nil
}

func (srv *authServiceImplementation) Logout(token string) error {
	return nil
}

func (srv *authServiceImplementation) Signup(login string, password string) (*thriftauth.AuthResponse, error) {
	return nil, nil
}

func (srv *authServiceImplementation) ValidateToken(token string) (*thriftauth.TokenValidationResponse, error) {
	return nil, nil
}

func main() {
	conf := DefaultConfig
	_, err := config.Parse(&conf)
	if err != nil {
		log.Fatal(err)
	}
	if conf.Environment == "" || (conf.Environment != "prod" && conf.Environment != "staging" && conf.Environment != "dev") {
		log.Fatal("flag --env is required and must be one of prod, staging, or dev")
	}

	authService := &authServiceImplementation{}
	serv := rpc.NewServer()
	if err := serv.RegisterName("Thrift", &thriftauth.AuthServer{Implementation: authService}); err != nil {
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
