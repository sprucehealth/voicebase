package config

import (
	"fmt"
	"net"
	"time"

	"carefront/libs/svcreg"
	"github.com/samuel/go-metrics/metrics"
)

type Server struct {
	Config          *BaseConfig
	ServiceID       svcreg.ServiceId
	ListenAddr      string
	ServFunc        func(net.Conn)
	MetricsRegistry metrics.Registry

	stopChan                   chan chan bool
	ln                         net.Listener
	statEstablishedConnections metrics.Counter
	statActiveConnections      metrics.IntegerGauge
}

func (ts *Server) Start() error {
	ln, err := net.Listen("tcp", ts.ListenAddr)
	if err != nil {
		return err
	}

	port := ln.Addr().(*net.TCPAddr).Port

	// Register the service

	addr, err := svcreg.Addr()
	if err != nil {
		return fmt.Errorf("Failed to get system's address: %+v", err)
	}
	svcMember := svcreg.Member{
		Endpoint:            svcreg.Endpoint{Host: addr, Port: port},
		AdditionalEndpoints: nil,
	}
	svcReg, err := ts.Config.ServiceRegistry()
	if err != nil {
		return err
	}
	reg, err := svcReg.Register(ts.ServiceID, svcMember)
	if err != nil {
		return fmt.Errorf("Failed to register services: %+v", err)
	}

	// Setup metrics

	ts.statEstablishedConnections = metrics.NewCounter()
	ts.statActiveConnections = metrics.NewIntegerGauge()
	ts.MetricsRegistry.Add("connections/established", ts.statEstablishedConnections)
	ts.MetricsRegistry.Add("connections/active", ts.statActiveConnections)

	// Start the service in a new Go routine

	ts.stopChan = make(chan chan bool, 1)
	ts.ln = ln

	go func() {
		defer reg.Unregister()
		for {
			select {
			case ch := <-ts.stopChan:
				ln.Close()
				if ch != nil {
					ch <- true
				}
				return
			default:
			}

			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			ts.statEstablishedConnections.Inc(1)
			go func() {
				ts.statActiveConnections.Inc(1)
				defer ts.statActiveConnections.Dec(1)
				ts.ServFunc(conn)
			}()
		}
	}()

	return nil
}

// Stop the service waiting a maximum amount of time. Return true if
// the service successfully stopped, otherwise return false on
// timeout.
func (ts *Server) Stop(wait time.Duration) bool {
	doneChan := make(chan bool, 1)
	ts.stopChan <- doneChan
	ts.ln.Close()
	if wait > 0 {
		select {
		case <-doneChan:
			return true
		case <-time.After(wait):
			return false
		}
	}
	return true
}
