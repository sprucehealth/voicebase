package zksvcreg

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"carefront/svcreg"
	"github.com/samuel/go-zookeeper/zk"
)

type registry struct {
	zkConn *zk.Conn
	prefix string

	mu         sync.Mutex
	watchers   map[svcreg.ServiceId]*watcher
	registered []*registeredService
}

type registeredService struct {
	reg      *registry
	id       svcreg.ServiceId
	member   svcreg.Member
	basePath string
	path     string
	stopCh   chan bool
}

type watcher struct {
	id       svcreg.ServiceId
	path     string
	mu       sync.Mutex
	reg      *registry
	members  map[string]svcreg.Member
	stopCh   chan bool
	channels []chan<- []svcreg.ServiceUpdate
}

func serviceIdToPath(prefix string, id svcreg.ServiceId) string {
	return strings.Join([]string{prefix, id.Environment, id.Name}, "/")
}

func (rs *registeredService) start() error {
	rs.stopCh = make(chan bool)
	data, err := json.MarshalIndent(rs.member, "", "    ")
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-rs.stopCh:
				return
			default:
			}

			path, err := rs.reg.zkConn.CreateProtectedEphemeralSequential(rs.basePath+"/member_", data, zk.WorldACL(zk.PermAll))
			if err != nil {
				log.Printf("zksvcreg: failed to create member node: %s", err.Error())
				time.Sleep(time.Second * 5)
				continue
			}

			_, stat, ch, err := rs.reg.zkConn.GetW(path)
			if err != nil {
				log.Printf("zksvcreg: error while monitoring registered service: %s", err.Error())
				continue
			}

			select {
			case <-rs.stopCh:
				if err := rs.reg.zkConn.Delete(path, stat.Version); err != nil {
					log.Printf("zksvcreg: failed to delete path during unregister of '%s': %+v", path, err)
				}
				return
			case ev := <-ch:
				if ev.Err != nil {
					log.Printf("zksvcreg: error while monitoring registered service: %s", ev.Err.Error())
				}
			}
		}
	}()

	return nil
}

func (rs *registeredService) stop() {
	close(rs.stopCh)
}

func (rs *registeredService) Unregister() error {
	rs.stop()
	rs.reg.removeRegistered(rs)
	return nil
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

func (w *watcher) start() error {
	if err := w.reg.createRecursive(w.path); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-w.stopCh:
				return
			default:
			}

			children, _, ch, err := w.reg.zkConn.ChildrenW(w.path)
			if err != nil {
				log.Printf("zksvcreg: failed to list children of %s: %+v", w.path, err)
				time.Sleep(time.Second * 5)
				continue
			}

			w.mu.Lock()
			var updates []svcreg.ServiceUpdate
			oldMembersList := w.members
			newMembersList := make(map[string]svcreg.Member, len(children))
			for _, name := range children {
				path := w.path + "/" + name
				data, _, err := w.reg.zkConn.Get(path)
				if err != nil {
					log.Printf("zksvcreg: failed to get contents of %s: %+v", path, err)
					continue
				}
				var mem svcreg.Member
				if err := json.Unmarshal(data, &mem); err != nil {
					log.Printf("zksvcreg: failed to unmarshal %s: %+v", path, err)
					continue
				}
				if _, ok := oldMembersList[name]; !ok {
					updates = append(updates, svcreg.ServiceUpdate{
						Type:   svcreg.Add,
						Member: mem,
					})
				} else {
					delete(oldMembersList, name)
				}
				newMembersList[name] = mem
			}
			if len(oldMembersList) > 0 {
				for _, mem := range oldMembersList {
					updates = append(updates, svcreg.ServiceUpdate{
						Type:   svcreg.Remove,
						Member: mem,
					})
				}
			}
			if len(updates) > 0 {
				for _, ch := range w.channels {
					select {
					case ch <- updates:
					default:
						log.Printf("zksvcreg: watcher channel full: %+v", ch)
					}
				}
			}
			w.members = newMembersList
			w.mu.Unlock()

			select {
			case ev := <-ch:
				if ev.Err != nil {
					log.Printf("zksvcreg: error watching service: %+v", ev.Err)
				}
			case <-w.stopCh:
				return
			}
		}
	}()

	return nil
}

func (w *watcher) stop() error {
	close(w.stopCh)
	return nil
}

func (w *watcher) addChannel(ch chan<- []svcreg.ServiceUpdate) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.channels = append(w.channels, ch)
	if len(w.members) > 0 {
		updates := make([]svcreg.ServiceUpdate, len(w.members))
		i := 0
		for _, m := range w.members {
			updates[i] = svcreg.ServiceUpdate{Type: svcreg.Add, Member: m}
			i++
		}
		select {
		case ch <- updates:
		default:
			log.Printf("zksvcreg: watcher channel full: %+v", ch)
		}
	}
	return nil
}

func (w *watcher) removeChannel(ch chan<- []svcreg.ServiceUpdate) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	for i, c := range w.channels {
		if c == ch {
			n := len(w.channels) - 1
			w.channels[i] = w.channels[n]
			w.channels[n] = nil // clear the reference for GC
			w.channels = w.channels[:n]
			break
		}
	}
	return nil
}
