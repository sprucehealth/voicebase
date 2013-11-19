package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"time"

	"carefront/libs/svcreg"
	"carefront/libs/svcreg/zksvcreg"
	"carefront/thriftauth"
	"github.com/samuel/go-thrift/thrift"
	"github.com/samuel/go-zookeeper/zk"
)

var (
	flagEnvironment             = flag.String("env", "", "REQUIRED: Environment (prod, staging, dev)")
	flagPort                    = flag.Int("port", 10001, "Port to bind to")
	flagZookeeperHosts          = flag.String("zookeeper_hosts", "", "Zookeeper host list (e.g. '127.0.0.1:2181,192.168.1.1:2181')")
	flagZookeeperServicesPrefix = flag.String("zookeeper_services_prefix", "/services", "Zookeeper service registry prefix")
)

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
	flag.Parse()
	if *flagEnvironment == "" || (*flagEnvironment != "prod" && *flagEnvironment != "staging" && *flagEnvironment != "dev") {
		log.Fatal("flag -env is required and must be one of prod, staging, or dev")
	}

	var reg svcreg.Registry

	var zoo *zk.Conn
	var zooCh <-chan zk.Event
	if *flagZookeeperHosts != "" {
		var err error
		hosts := strings.Split(*flagZookeeperHosts, ",")
		zoo, zooCh, err = zk.Connect(hosts, time.Second*10)
		if err != nil {
			log.Fatal(err)
		}
		defer zoo.Close()
		reg, err = zksvcreg.NewServiceRegistry(zoo, *flagZookeeperServicesPrefix)
		if err != nil {
			log.Fatal(err)
		}
		defer reg.Close()
	} else {
		reg = &svcreg.StaticRegistry{}
	}
	_ = zooCh

	authService := &authServiceImplementation{}
	rpc.RegisterName("Thrift", &thriftauth.AuthServer{authService})

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *flagPort))
	if err != nil {
		log.Fatal(err)
	}

	addr, err := svcreg.Addr()
	if err != nil {
		log.Fatalf("Failed to get system's address: %+v", err)
	}
	svcId := svcreg.ServiceId{Environment: *flagEnvironment, Name: serviceName}
	svcMember := svcreg.Member{
		Endpoint:            svcreg.Endpoint{Host: addr, Port: *flagPort},
		AdditionalEndpoints: nil,
	}
	svcReg, err := reg.Register(svcId, svcMember)
	if err != nil {
		log.Fatalf("Failed to register services: %+v", err)
	}
	defer svcReg.Unregister()

	stopChan := make(chan bool)

	go func() {
		for {
			select {
			case <-stopChan:
				ln.Close()
				return
			default:
			}

			conn, err := ln.Accept()
			if err != nil {
				log.Printf("Accept failed: %+v\n", err)
				continue
			}
			go rpc.ServeCodec(thrift.NewServerCodec(thrift.NewFramedReadWriteCloser(conn, 0), thrift.NewBinaryProtocol(true, false)))
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	_ = <-signalChan
	close(stopChan)
	time.Sleep(time.Second * 1)
}
