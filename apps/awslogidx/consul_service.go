package main

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/armon/consul-api"
)

const (
	consulCheckTTL  = "60s"
	consulLockDelay = time.Second * 30
)

type Service struct {
	id, name   string
	tags       []string
	port       int
	consul     *consulapi.Client
	checkID    string
	checkQuitC chan chan bool
	sessionID  string
	mu         sync.Mutex
	locks      map[string]*Lock
}

type Lock struct {
	locked int32
	svc    *Service
	key    string
	value  []byte
	stopCh chan bool
	log    golog.Logger
}

func RegisterService(consul *consulapi.Client, id, name string, tags []string, port int) (*Service, error) {
	if id == "" {
		id = name
	}
	if err := consul.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
		ID:   id,
		Name: name,
		Tags: tags,
		Port: port,
		Check: &consulapi.AgentServiceCheck{
			TTL: consulCheckTTL,
		},
	}); err != nil {
		return nil, err
	}

	s := &Service{
		id:      id,
		name:    name,
		tags:    tags,
		port:    port,
		consul:  consul,
		checkID: "service:" + id, // This is implied id by Consul when creating a service
		locks:   make(map[string]*Lock),
	}

	s.startChecker()
	if err := s.createSession(); err != nil {
		s.stopChecker()
		s.Deregister()
		return nil, err
	}

	return s, nil
}

func (s *Service) Deregister() error {
	s.mu.Lock()
	for _, lock := range s.locks {
		lock.release()
	}
	s.locks = nil
	s.mu.Unlock()

	s.destroySession()
	s.stopChecker()
	return s.consul.Agent().ServiceDeregister(s.id)
}

func (s *Service) CheckID() string {
	return s.checkID
}

func (s *Service) startChecker() {
	// Initial check
	if err := s.consul.Agent().PassTTL(s.checkID, "Startup"); err != nil {
		golog.Errorf("Failed to update check TTL: %s", err.Error())
	}

	s.checkQuitC = make(chan chan bool, 1)
	go func() {
		t := time.NewTicker(time.Second * 30)
		defer func() {
			t.Stop()
		}()

		for {
			select {
			case ch := <-s.checkQuitC:
				if err := s.consul.Agent().CheckDeregister(s.checkID); err != nil {
					golog.Errorf("Failed to deregister consul check: %s", err.Error())
				}
				ch <- true
				return
			case tm := <-t.C:
				if err := s.consul.Agent().PassTTL(s.checkID, tm.String()); err != nil {
					golog.Errorf("Failed to update check TTL: %s", err.Error())
				}
			}
		}
	}()
}

func (s *Service) stopChecker() {
	if s.checkQuitC != nil {
		ch := make(chan bool)
		s.checkQuitC <- ch
		select {
		case <-ch:
			s.checkQuitC = nil
		case <-time.After(time.Second * 5):
		}
	}
}

func (s *Service) createSession() error {
	sessionID, _, err := s.consul.Session().Create(&consulapi.SessionEntry{
		LockDelay: consulLockDelay,
		Checks: []string{
			"serfHealth", // Default health check for consul process liveliness
			s.checkID,
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create session: %s", err.Error())
	}
	s.sessionID = sessionID
	return nil
}

func (s *Service) destroySession() error {
	if s.sessionID != "" {
		if _, err := s.consul.Session().Destroy(s.sessionID, nil); err != nil {
			return fmt.Errorf("Failed to destroy consul session: %s", err.Error())
		}
		s.sessionID = ""
	}
	return nil
}

func (s *Service) removeLock(key string) {
	s.mu.Lock()
	delete(s.locks, key)
	s.mu.Unlock()
}

func (s *Service) NewLock(key string, value []byte) *Lock {
	if value == nil {
		value = []byte(s.sessionID)
	}
	lock := &Lock{
		svc:    s,
		key:    key,
		value:  value,
		stopCh: make(chan bool, 1),
		log:    golog.Context("key", key, "session", s.sessionID),
	}
	s.mu.Lock()
	s.locks[key] = lock
	s.mu.Unlock()
	lock.start()
	return lock
}

func (l *Lock) Locked() bool {
	return atomic.LoadInt32(&l.locked) != 0
}

func (l *Lock) Wait() bool {
	for !l.Locked() {
		select {
		case <-l.stopCh:
			return false
		default:
		}
		time.Sleep(time.Second)
	}
	return true
}

func (l *Lock) Release() {
	l.release()
	l.svc.removeLock(l.key)
}

// release stops the lock loop without removing the lock from the parent locker
func (l *Lock) release() {
	close(l.stopCh)
}

func (l *Lock) setLocked(b bool) {
	if b {
		atomic.StoreInt32(&l.locked, 1)
	} else {
		atomic.StoreInt32(&l.locked, 0)
	}
}

func (l *Lock) start() {
	go func() {
		for {
			select {
			case <-l.stopCh:
				return
			default:
			}
			if leader, _, err := l.svc.consul.KV().Acquire(&consulapi.KVPair{
				Key:     l.key,
				Value:   l.value,
				Session: l.svc.sessionID,
			}, nil); err != nil {
				if strings.Contains(err.Error(), "Invalid session") {
					l.log.Fatalf("Invalid session: %s", err.Error())
				}
				l.log.Errorf("Error acquiring lock: %s", err.Error())
			} else {
				l.setLocked(leader)
				if leader {
					l.log.Infof("Became leader")
				} else {
					l.log.Infof("Not leader")
				}

				var lastIndex uint64
				for {
					kv, meta, err := l.svc.consul.KV().Get(l.key, &consulapi.QueryOptions{
						WaitIndex: lastIndex,
					})
					if err != nil {
						// Assume we're not the leader for now since it's safer.
						l.setLocked(false)
						l.log.Errorf("Failed to get leader key (dropping leadership): %s", err.Error())
						time.Sleep(time.Second * 10)
						lastIndex = 0
						continue
					}
					select {
					case <-l.stopCh:
						return
					default:
					}

					lastIndex = meta.LastIndex
					if kv == nil || kv.Session == "" {
						l.log.Infof("No leader. Attempting to take power after %s", time.Duration(consulLockDelay).String())
						l.setLocked(false)
						break
					} else if kv.Session == l.svc.sessionID {
						if !l.Locked() {
							// This should only happen if there was previously an error
							// talking to consul.
							l.setLocked(true)
							l.log.Warningf("Remembering own leadership")
						}
						continue
					}
					if l.Locked() {
						l.setLocked(false)
						l.log.Warningf("Lost leadership to %s", kv.Session)
					} else {
						l.log.Infof("Current leader is %s", kv.Session)
					}
				}
			}
			// After the lock is released there's a period of time before which
			// it can be acquired. This allows for a process that involuntarily
			// lost the lock to notice they lost the lock and complete processing
			// before another process becomes leader.
			time.Sleep(consulLockDelay)
		}
	}()
}
