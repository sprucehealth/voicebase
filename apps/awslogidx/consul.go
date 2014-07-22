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
	consulCheckIDPrefix = "awslogidx-"
	consulCheckName     = "Liveness check for awslogidx process"
	consulCheckTTL      = "60s"
	// consulLockDelay is the time after a lock is release before it can be acquired
	consulLockDelay = time.Second * 30
)

type ConsulLocker struct {
	cli        *consulapi.Client
	checkID    string
	checkQuitC chan chan bool
	sessionID  string
	mu         sync.Mutex
	locks      map[string]*ConsulLock
}

type ConsulLock struct {
	locked int32
	cl     *ConsulLocker
	key    string
	value  []byte
	stopCh chan bool
	log    golog.Logger
}

func StartConsulLocker(cli *consulapi.Client, checkID string) (*ConsulLocker, error) {
	cl := &ConsulLocker{
		cli:     cli,
		checkID: checkID,
		locks:   make(map[string]*ConsulLock),
	}
	if err := cl.startCheck(); err != nil {
		return nil, err
	}
	if err := cl.createSession(); err != nil {
		cl.stopCheck()
		return nil, err
	}
	return cl, nil
}

func (c *ConsulLocker) Stop() {
	c.mu.Lock()
	for _, lock := range c.locks {
		lock.release()
	}
	c.locks = nil
	c.mu.Unlock()

	if err := c.destroySession(); err != nil {
		golog.Errorf(err.Error())
	}

	if err := c.stopCheck(); err != nil {
		golog.Errorf(err.Error())
	}
}

func (cl *ConsulLocker) startCheck() error {
	// The check may have been left registered from a previous run that crashes in the rare
	// case that this process happens to get the same pid. In this case deregister the old
	// check which will cause the old session to be invalidated. If we don't deregister then
	// the old session could remain valid if we started to use the same check. Unused checks will
	// be left around if the process crashes, but this should hopefully be rare and can be cleaned
	// up by listing check IDs and seeing if the pid is valid.
	for retries := 0; ; retries++ {
		if err := cl.cli.Agent().CheckRegister(&consulapi.AgentCheckRegistration{
			ID:   cl.checkID,
			Name: consulCheckName,
			AgentServiceCheck: consulapi.AgentServiceCheck{
				TTL: consulCheckTTL,
			},
		}); err != nil {
			if !strings.Contains(err.Error(), "already registered") {
				return fmt.Errorf("failed to register consul check: %s", err.Error())
			}
			if err := cl.cli.Agent().CheckDeregister(cl.checkID); err != nil {
				return fmt.Errorf("failed to deregister old consul check: %s", err.Error())
			}
		} else {
			break
		}
	}
	cl.checkQuitC = make(chan chan bool, 1)

	go func() {
		t := time.NewTicker(time.Second * 30)
		defer func() {
			t.Stop()
		}()

		for {
			select {
			case ch := <-cl.checkQuitC:
				if err := cl.cli.Agent().CheckDeregister(cl.checkID); err != nil {
					golog.Errorf("Failed to deregister consul check: %s", err.Error())
				}
				ch <- true
				return
			case tm := <-t.C:
				if err := cl.cli.Agent().PassTTL(cl.checkID, tm.String()); err != nil {
					golog.Errorf("Failed to update check TTL: %s", err.Error())
				}
			}
		}
	}()

	return nil
}

func (cl *ConsulLocker) stopCheck() error {
	if cl.checkQuitC != nil {
		ch := make(chan bool)
		cl.checkQuitC <- ch
		select {
		case <-ch:
			cl.checkQuitC = nil
		case <-time.After(time.Second * 5):
		}
	}
	return nil
}

func (cl *ConsulLocker) createSession() error {
	sessionID, _, err := cl.cli.Session().Create(&consulapi.SessionEntry{
		LockDelay: consulLockDelay,
		Checks: []string{
			"serfHealth", // Default health check for consul process liveliness
			cl.checkID,
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create session: %s", err.Error())
	}
	cl.sessionID = sessionID
	return nil
}

func (cl *ConsulLocker) destroySession() error {
	if cl.sessionID != "" {
		if _, err := cl.cli.Session().Destroy(cl.sessionID, nil); err != nil {
			return fmt.Errorf("Failed to destroy consul session: %s", err.Error())
		}
		cl.sessionID = ""
	}
	return nil
}

func (cl *ConsulLocker) removeLock(key string) {
	cl.mu.Lock()
	delete(cl.locks, key)
	cl.mu.Unlock()
}

func (cl *ConsulLocker) NewLock(key string, value []byte) *ConsulLock {
	if value == nil {
		value = []byte(cl.sessionID)
	}
	lock := &ConsulLock{
		cl:     cl,
		key:    key,
		value:  value,
		stopCh: make(chan bool, 1),
		log:    golog.Context("key", key, "session", cl.sessionID),
	}
	cl.mu.Lock()
	cl.locks[key] = lock
	cl.mu.Unlock()
	lock.start()
	return lock
}

func (cl *ConsulLock) Locked() bool {
	return atomic.LoadInt32(&cl.locked) != 0
}

func (cl *ConsulLock) Wait() bool {
	for !cl.Locked() {
		select {
		case <-cl.stopCh:
			return false
		default:
		}
		time.Sleep(time.Second)
	}
	return true
}

func (cl *ConsulLock) Release() {
	cl.release()
	cl.cl.removeLock(cl.key)
}

// release stops the lock loop without removing the lock from the parent locker
func (cl *ConsulLock) release() {
	close(cl.stopCh)
}

func (cl *ConsulLock) setLocked(b bool) {
	if b {
		atomic.StoreInt32(&cl.locked, 1)
	} else {
		atomic.StoreInt32(&cl.locked, 0)
	}
}

func (cl *ConsulLock) start() {
	go func() {
		for {
			select {
			case <-cl.stopCh:
				return
			default:
			}
			if leader, _, err := cl.cl.cli.KV().Acquire(&consulapi.KVPair{
				Key:     cl.key,
				Value:   cl.value,
				Session: cl.cl.sessionID,
			}, nil); err != nil {
				if strings.Contains(err.Error(), "Invalid session") {
					cl.log.Fatalf("Invalid session: %s", err.Error())
				}
				cl.log.Errorf("Error acquiring lock: %s", err.Error())
			} else {
				cl.setLocked(leader)
				if leader {
					cl.log.Infof("Became leader")
				} else {
					cl.log.Infof("Not leader")
				}

				var lastIndex uint64
				for {
					kv, meta, err := cl.cl.cli.KV().Get(cl.key, &consulapi.QueryOptions{
						WaitIndex: lastIndex,
					})
					if err != nil {
						// Assume we're not the leader for now since it's safer.
						cl.setLocked(false)
						cl.log.Errorf("Failed to get leader key (dropping leadership): %s", err.Error())
						time.Sleep(time.Second * 10)
						lastIndex = 0
						continue
					}
					select {
					case <-cl.stopCh:
						return
					default:
					}

					lastIndex = meta.LastIndex
					if kv == nil || kv.Session == "" {
						cl.log.Infof("No leader. Attempting to take power after %s", time.Duration(consulLockDelay).String())
						cl.setLocked(false)
						break
					} else if kv.Session == cl.cl.sessionID {
						if !cl.Locked() {
							// This should only happen if there was previously an error
							// talking to consul.
							cl.setLocked(true)
							cl.log.Warningf("Remembering own leadership")
						}
						continue
					}
					if cl.Locked() {
						cl.setLocked(false)
						cl.log.Warningf("Lost leadership to %s", kv.Session)
					} else {
						cl.log.Infof("Current leader is %s", kv.Session)
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
