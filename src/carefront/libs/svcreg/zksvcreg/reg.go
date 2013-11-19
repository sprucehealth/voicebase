package zksvcreg

import (
	"errors"
	"strings"
	"sync"

	"carefront/libs/svcreg"
	"github.com/samuel/go-zookeeper/zk"
)

type registry struct {
	zkConn *zk.Conn
	prefix string

	mu         sync.Mutex
	watchers   map[svcreg.ServiceId]*watcher
	registered []*registeredService
}

func serviceIdToPath(prefix string, id svcreg.ServiceId) string {
	return strings.Join([]string{prefix, id.Environment, id.Name}, "/")
}

func NewServiceRegistry(zkConn *zk.Conn, pathPrefix string) (svcreg.Registry, error) {
	if zkConn == nil {
		return nil, errors.New("zksvcreg: zkConn must not be nil")
	}

	if pathPrefix == "" {
		pathPrefix = "/services/"
	} else if pathPrefix[0] != '/' {
		pathPrefix = "/" + pathPrefix
	}
	if pathPrefix[len(pathPrefix)-1] == '/' {
		pathPrefix = pathPrefix[:len(pathPrefix)-1]
	}

	reg := &registry{
		zkConn: zkConn,
		prefix: pathPrefix,
	}

	if err := reg.createRecursive(pathPrefix); err != nil {
		return nil, err
	}

	return reg, nil
}

func (r *registry) createRecursive(path string) error {
	if exists, _, err := r.zkConn.Exists(path); err != nil {
		return err
	} else if exists {
		return nil
	}

	parts := strings.Split(path, "/")
	for i := 2; i <= len(parts); i++ {
		curPath := strings.Join(parts[:i], "/")
		if _, err := r.zkConn.Create(curPath, []byte{}, 0, zk.WorldACL(zk.PermAll)); err != nil && err != zk.ErrNodeExists {
			return err
		}
	}

	return nil
}

func (r *registry) WatchService(id svcreg.ServiceId, watchCh chan<- []svcreg.ServiceUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.watchers == nil {
		r.watchers = make(map[svcreg.ServiceId]*watcher)
	}
	watcher := r.watchers[id]
	if watcher == nil {
		var err error
		watcher, err = r.newWatcher(id)
		if err != nil {
			return err
		}
		r.watchers[id] = watcher
	}
	return watcher.addChannel(watchCh)
}

func (r *registry) UnwatchService(id svcreg.ServiceId, watchCh chan<- []svcreg.ServiceUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.watchers == nil {
		return nil
	}
	if w := r.watchers[id]; w != nil {
		// TODO: could destroy the watcher if all channels removed
		return w.removeChannel(watchCh)
	}
	return nil
}

func (r *registry) Register(id svcreg.ServiceId, member svcreg.Member) (svcreg.RegisteredService, error) {
	path := serviceIdToPath(r.prefix, id)
	if err := r.createRecursive(path); err != nil {
		return nil, err
	}
	rs := &registeredService{
		reg:      r,
		id:       id,
		member:   member,
		basePath: path,
	}
	return rs, rs.start()
}

func (r *registry) removeRegistered(rs *registeredService) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, s := range r.registered {
		if s == rs {
			n := len(r.registered) - 1
			r.registered[i] = r.registered[n]
			r.registered[n] = nil // clear the reference for GC
			r.registered = r.registered[:n]
			break
		}
	}
}

func (r *registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, w := range r.watchers {
		w.stop()
	}
	r.watchers = nil
	for _, rs := range r.registered {
		rs.stop()
	}
	r.registered = nil
	return nil
}

func (r *registry) newWatcher(id svcreg.ServiceId) (*watcher, error) {
	w := &watcher{
		id:     id,
		path:   serviceIdToPath(r.prefix, id),
		reg:    r,
		stopCh: make(chan bool),
	}
	return w, w.start()
}
