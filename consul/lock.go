package consul

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/armon/consul-api"
	"github.com/sprucehealth/backend/libs/golog"
)

type Lock struct {
	locked int32
	delay  time.Duration
	svc    *Service
	key    string
	value  []byte
	stopCh chan chan bool
	log    golog.Logger
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
	l.stop()
	l.svc.removeLock(l.key)
}

func (l *Lock) stop() {
	ch := make(chan bool, 1)
	l.stopCh <- ch
	select {
	case <-ch:
	case <-time.After(time.Second):
		golog.Errorf("Timeout waiting for lock release")
	}
	close(l.stopCh)
}

func (l *Lock) checkStop() bool {
	select {
	case ch := <-l.stopCh:
		select {
		case ch <- true:
		default:
		}
		return true
	default:
	}
	return false
}

func (l *Lock) sleep(waitSec int) bool {
	select {
	case ch := <-l.stopCh:
		select {
		case ch <- true:
		default:
		}
		return true
	case <-time.After(time.Second * time.Duration(waitSec)):
	}
	return false
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
		qo := &consulapi.QueryOptions{
			WaitIndex: 0,
		}
		var sessionID string
		defer func() {
			if sessionID != "" {
				if err := l.svc.destroySession(sessionID); err != nil {
					l.log.Errorf("Failed to destroy session %s: %s", sessionID, err.Error())
				}
			}
		}()
		for !l.checkStop() {
			if sessionID == "" {
				var err error
				sessionID, err = l.svc.createSession(l.delay)
				if err != nil {
					l.log.Errorf("Failed to create session: %s", err.Error())
					if l.sleep(5) {
						return
					}
					continue
				}
			}
			log := l.log.Context("session", sessionID)

			if leader, _, err := l.svc.consul.KV().Acquire(&consulapi.KVPair{
				Key:     l.key,
				Value:   l.value,
				Session: sessionID,
			}, nil); err != nil {
				if strings.Contains(err.Error(), "Invalid session") {
					sessionID = ""
					log.Errorf("Invalid session: %s", err.Error())
					if l.sleep(5) {
						return
					}
					continue
				}
				log.Errorf("Error acquiring lock: %s", err.Error())
			} else {
				l.setLocked(leader)
				if leader {
					log.Infof("Lock aquired")
				} else {
					log.Infof("Lock not aquired")
				}

				for {
					kv, meta, err := l.svc.consul.KV().Get(l.key, qo)
					if err != nil {
						// Assume we're not the leader for now since it's safer.
						l.setLocked(false)
						log.Errorf("Failed to get leader key (dropping leadership): %s", err.Error())
						if l.sleep(5) {
							return
						}
						qo.WaitIndex = 0
						continue
					}
					if l.checkStop() {
						return
					}

					qo.WaitIndex = meta.LastIndex
					if kv == nil || kv.Session == "" {
						log.Infof("No leader. Attempting to take power after %s", l.delay.String())
						l.setLocked(false)
						break
					} else if kv.Session == sessionID {
						if !l.Locked() {
							// This should only happen if there was previously an error
							// talking to consul.
							l.setLocked(true)
							log.Warningf("Remembering own leadership")
						}
						continue
					}
					if l.Locked() {
						l.setLocked(false)
						log.Warningf("Lost leadership to %s", kv.Session)
					} else {
						log.Infof("Current leader is %s", kv.Session)
					}
				}
			}
			// After the lock is released there's a period of time before which
			// it can be acquired. This allows for a process that involuntarily
			// lost the lock to notice they lost the lock and complete processing
			// before another process becomes leader.
			time.Sleep(l.delay)
		}
	}()
}
