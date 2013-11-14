package zksvcreg

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"carefront/svcreg"
	"github.com/samuel/go-zookeeper/zk"
)

type watcher struct {
	id       svcreg.ServiceId
	path     string
	mu       sync.Mutex
	reg      *registry
	members  map[string]svcreg.Member
	stopCh   chan bool
	channels []chan<- []svcreg.ServiceUpdate
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
				log.Printf("zksvcreg/watcher: failed to list children of %s: %+v", w.path, err)
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
					if err != zk.ErrNoNode {
						log.Printf("zksvcreg/watcher: failed to get contents of %s: %+v", path, err)
					}
					continue
				}
				var mem svcreg.Member
				if err := json.Unmarshal(data, &mem); err != nil {
					log.Printf("zksvcreg/watcher: failed to unmarshal %s: %+v", path, err)
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
				// Flip to have removes before adds
				for i := 0; i < len(updates)/2; i++ {
					i2 := len(updates) - 1 - i
					t := updates[i]
					updates[i] = updates[i2]
					updates[i2] = t
				}
			}
			if len(updates) > 0 {
				for _, ch := range w.channels {
					select {
					case ch <- updates:
					default:
						log.Printf("zksvcreg/watcher: watcher channel full: %+v", ch)
					}
				}
			}
			w.members = newMembersList
			w.mu.Unlock()

			select {
			case ev := <-ch:
				if ev.Err != nil {
					log.Printf("zksvcreg/watcher: error watching service: %+v", ev.Err)
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
			log.Printf("zksvcreg/watcher: watcher channel full: %+v", ch)
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
