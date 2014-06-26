package svcclient

import (
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"sync"

	"github.com/sprucehealth/backend/libs/svcreg"

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-thrift/thrift"
)

type ThriftServiceClientBuilder struct {
	registry  svcreg.Registry
	serviceId svcreg.ServiceId
	hosts     []string
	updateCh  chan []svcreg.ServiceUpdate
	stopCh    chan chan bool
	mu        sync.Mutex
}

func NewThriftServiceClientBuilder(reg svcreg.Registry, id svcreg.ServiceId) (*ThriftServiceClientBuilder, error) {
	scb := &ThriftServiceClientBuilder{
		registry:  reg,
		serviceId: id,
		hosts:     make([]string, 0, 8),
		updateCh:  make(chan []svcreg.ServiceUpdate, 8),
		stopCh:    make(chan chan bool, 1),
	}
	scb.start()
	if err := reg.WatchService(id, scb.updateCh); err != nil {
		scb.Stop()
		return nil, err
	}
	return scb, nil
}

func (sb *ThriftServiceClientBuilder) start() {
	go func() {
		for {
			select {
			case ch := <-sb.stopCh:
				ch <- true
				return
			case updates := <-sb.updateCh:
				sb.mu.Lock()
				for _, up := range updates {
					host := up.Member.Endpoint.String()
					if up.Type == svcreg.Add {
						sb.hosts = append(sb.hosts, host)
					} else if up.Type == svcreg.Remove {
						for i := 0; i < len(sb.hosts); i++ {
							if sb.hosts[i] == host {
								sb.hosts[i] = sb.hosts[len(sb.hosts)-1]
								sb.hosts = sb.hosts[:len(sb.hosts)-1]
							}
						}
					}
				}
				sb.mu.Unlock()
			}
		}
	}()
}

func (sb *ThriftServiceClientBuilder) Stop() {
	ch := make(chan bool, 1)
	sb.stopCh <- ch
	<-ch
}

func (sb *ThriftServiceClientBuilder) NewClient() (*rpc.Client, error) {
	sb.mu.Lock()
	if len(sb.hosts) == 0 {
		sb.mu.Unlock()
		return nil, fmt.Errorf("svcclient: no hosts for service %s/%s", sb.serviceId.Environment, sb.serviceId.Name)
	}
	addr := sb.hosts[rand.Intn(len(sb.hosts))]
	sb.mu.Unlock()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return thrift.NewClient(thrift.NewFramedReadWriteCloser(conn, 0), thrift.NewBinaryProtocol(true, false), false), nil
}
