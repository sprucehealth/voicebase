package consul

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	consulCheckTTL = "60s"
)

// Service defines a service that is made available over a consul cluster.
// It can contain distributed locks created within a session.
// Following links provide an understanding of the different concepts at play:
// - Consul: 			http://www.consul.io/docs/internals/architecture.html
// - Checks:			http://www.consul.io/docs/agent/checks.html
// - Sessions: 			http://www.consul.io/docs/internals/sessions.html
// - Leader election:	http://www.consul.io/docs/guides/leader-election.html
type Service struct {
	isRegistered uint32 // flag to indicate a valid registration (atomic)
	id, name     string
	tags         []string
	port         int
	consul       *consulapi.Client
	checkID      string
	stopCh       chan chan bool
	mu           sync.Mutex
	locks        map[string]*Lock
	log          golog.Logger
}

func RegisterService(consul *consulapi.Client, id, name string, tags []string, port int) (*Service, error) {
	if id == "" {
		id = name
	}

	s := &Service{
		id:      id,
		name:    name,
		tags:    tags,
		port:    port,
		consul:  consul,
		checkID: "service:" + id, // This is the implicit id creating a service
		locks:   make(map[string]*Lock),
		stopCh:  make(chan chan bool, 1),
		log:     golog.Context("id", id),
	}

	go s.loop()
	return s, nil
}

func (s *Service) Deregister() error {
	ch := make(chan bool, 1)
	s.stopCh <- ch
	select {
	case <-ch:
	case <-time.After(time.Second * 5):
		s.log.Errorf("Timeout waiting to degister service")
	}
	close(s.stopCh)
	return nil
}

func (s *Service) CheckID() string {
	return s.checkID
}

// WaitForRegistration returns once a the service is registered or the
// timeout is reached. If a valid registration exists then return
// true, otherwise if timeout is reached then return false.
func (s *Service) WaitForRegistration(timeout time.Duration) bool {
	// TODO: for now this is a simple busy/sleep loop waiting for isRegistered to
	// become 1. It could be improved to use signaling (channels or sync.Cond or
	// something), but it's probably not necessary since the registration should be
	// nearly always valid or registered relatively quickly if not.

	// Be lazy about getting the current time to avoid a syscall
	var startTime time.Time
	for atomic.LoadUint32(&s.isRegistered) == 0 {
		now := time.Now()
		if startTime.IsZero() {
			startTime = now
		} else if now.Sub(startTime) >= timeout {
			return false
		}
		time.Sleep(time.Millisecond * 100)
	}
	return true
}

func (s *Service) loop() {
	defer func() {
		s.mu.Lock()
		for _, lock := range s.locks {
			lock.stop()
		}
		s.locks = nil
		s.mu.Unlock()
		s.deregisterService()
	}()
	for !s.checkStop() {
		atomic.StoreUint32(&s.isRegistered, 0)

		// Try to deregister the service to force any old sessions to be invalidated
		if err := s.deregisterService(); err != nil {
			s.log.Errorf("Failed to deregister service: %s", err.Error())
			if s.sleep(5) {
				return
			}
			continue
		}
		if err := s.registerService(); err != nil {
			s.log.Errorf("Failed to register service: %s", err.Error())
			if s.sleep(5) {
				return
			}
			continue
		}
		golog.Infof("Registered service %s", s.id)

		atomic.StoreUint32(&s.isRegistered, 1)

		for {
			if s.sleep(5) {
				return
			}
			if err := s.consul.Agent().PassTTL(s.checkID, ""); err != nil {
				s.log.Errorf("Failed to update check TTL: %s", err.Error())
				if strings.Contains(err.Error(), "CheckID does not have associated TTL") {
					break
				}
			}
		}
	}
}

func (s *Service) checkStop() bool {
	select {
	case ch := <-s.stopCh:
		select {
		case ch <- true:
		default:
		}
		return true
	default:
	}
	return false
}

func (s *Service) sleep(waitSec int) bool {
	select {
	case ch := <-s.stopCh:
		select {
		case ch <- true:
		default:
		}
		return true
	case <-time.After(time.Second * time.Duration(waitSec)):
	}
	return false
}

func (s *Service) registerService() error {
	return s.consul.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
		ID:   s.id,
		Name: s.name,
		Tags: s.tags,
		Port: s.port,
		Check: &consulapi.AgentServiceCheck{
			TTL: consulCheckTTL,
		},
	})
}

func (s *Service) deregisterService() error {
	return s.consul.Agent().ServiceDeregister(s.id)
}

func (s *Service) createSession(lockDelay time.Duration) (string, error) {
	sessionID, _, err := s.consul.Session().Create(&consulapi.SessionEntry{
		Name:      fmt.Sprintf("Lock for %s service", s.name),
		LockDelay: lockDelay,
		Checks: []string{
			"serfHealth", // Default health check for consul process liveliness
			s.checkID,
		},
	}, nil)
	return sessionID, err
}

func (s *Service) destroySession(sessionID string) error {
	_, err := s.consul.Session().Destroy(sessionID, nil)
	return err
}

func (s *Service) removeLock(key string) {
	s.mu.Lock()
	delete(s.locks, key)
	s.mu.Unlock()
}

func (s *Service) NewLock(key string, value []byte, delay time.Duration) *Lock {
	s.mu.Lock()
	lock := &Lock{
		delay:  delay,
		svc:    s,
		key:    key,
		value:  value,
		stopCh: make(chan chan bool, 1),
		log:    golog.Context("key", key),
	}
	s.locks[key] = lock
	s.mu.Unlock()
	lock.start()
	return lock
}
