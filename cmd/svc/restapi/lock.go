package main

import (
	"sync"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/consul"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
)

type localLock struct {
	mu         sync.Mutex
	internalmu sync.Mutex
	isLocked   bool
}

func newLocalLock() api.LockAPI {
	return &localLock{}
}

func (l *localLock) Wait() bool {
	l.internalmu.Lock()
	defer l.internalmu.Unlock()
	l.mu.Lock()
	l.isLocked = true
	return true
}

func (l *localLock) Release() {
	l.internalmu.Lock()
	defer l.internalmu.Unlock()
	l.mu.Unlock()
	l.isLocked = false
}

func (l *localLock) Locked() bool {
	l.internalmu.Lock()
	defer l.internalmu.Unlock()
	return l.isLocked
}

func newConsulLock(name string, consulService *consul.Service, isDebug bool, gologHandler golog.Handler) api.LockAPI {
	var lock api.LockAPI
	log := golog.Context("key", name)
	log.SetHandler(gologHandler)
	if consulService != nil {
		lock = consulService.NewLock(name, nil, time.Second*30, log)
	} else if isDebug || environment.IsDemo() || environment.IsDev() {
		lock = newLocalLock()
	} else {
		golog.Fatalf("Unable to setup lock due to lack of consul service")
	}

	return lock
}
