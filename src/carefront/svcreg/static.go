package svcreg

import (
	"errors"
	"sync"
)

type StaticRegistry struct {
	Services map[ServiceId]map[Endpoint]Member

	watchers map[ServiceId][]chan<- []ServiceUpdate
	mu       sync.Mutex
}

type staticRegisteredService struct {
	reg    *StaticRegistry
	id     ServiceId
	member Member
}

func (s *staticRegisteredService) Unregister() error {
	s.reg.mu.Lock()
	defer s.reg.mu.Unlock()
	delete(s.reg.Services[s.id], s.member.Endpoint)
	if wat := s.reg.watchers[s.id]; wat != nil {
		update := ServiceUpdate{
			Type:   Remove,
			Member: s.member,
		}
		for _, ch := range wat {
			select {
			case ch <- []ServiceUpdate{update}:
			default:
			}
		}
	}
	return nil
}

func (r *StaticRegistry) WatchService(id ServiceId, watchCh chan<- []ServiceUpdate) error {
	if id.Environment == "" || id.Name == "" {
		return errors.New("svcreg: service environment or name not set")
	}
	if watchCh == nil {
		return errors.New("svcreg: watch channel may not be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.watchers == nil {
		r.watchers = make(map[ServiceId][]chan<- []ServiceUpdate)
	}

	wat := r.watchers[id]
	if wat == nil {
		wat = make([]chan<- []ServiceUpdate, 0, 4)
		r.watchers[id] = wat
	}
	for _, ch := range wat {
		if ch == watchCh {
			return errors.New("svcreg: already watching service on channel")
		}
	}
	r.watchers[id] = append(wat, watchCh)

	if members := r.Services[id]; members != nil {
		updates := make([]ServiceUpdate, len(members))
		i := 0
		for _, mem := range members {
			updates[i] = ServiceUpdate{
				Type:   Add,
				Member: mem,
			}
			i++
		}
		select {
		case watchCh <- updates:
		default:
		}
	}

	return nil
}

func (r *StaticRegistry) UnwatchService(id ServiceId, watchCh chan<- []ServiceUpdate) error {
	if id.Environment == "" || id.Name == "" {
		return errors.New("svcreg: service environment or name not set")
	}
	if watchCh == nil {
		return errors.New("svcreg: watch channel may not be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.watchers == nil {
		r.watchers = make(map[ServiceId][]chan<- []ServiceUpdate)
	}

	wat := r.watchers[id]
	if wat == nil {
		return errors.New("svcreg: channel not watching service")
	}

	for i, ch := range wat {
		if ch == watchCh {
			n := len(wat) - 1
			wat[i] = wat[n]
			wat[n] = nil // clear the reference for GC
			r.watchers[id] = wat[:n]
			return nil
		}
	}

	return errors.New("svcreg: channel not watching service")
}

func (r *StaticRegistry) Register(id ServiceId, member Member) (RegisteredService, error) {
	if member.Endpoint.Host == "" || member.Endpoint.Port == 0 {
		return nil, errors.New("svcreg: endpoint host or port not set")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Services == nil {
		r.Services = make(map[ServiceId]map[Endpoint]Member)
	}

	members := r.Services[id]
	if members == nil {
		members = make(map[Endpoint]Member)
		r.Services[id] = members
	}

	if _, ok := members[member.Endpoint]; ok {
		return nil, errors.New("svcreg: already registered")
	}
	members[member.Endpoint] = member

	if wat := r.watchers[id]; wat != nil {
		update := ServiceUpdate{
			Type:   Add,
			Member: member,
		}
		for _, ch := range wat {
			select {
			case ch <- []ServiceUpdate{update}:
			default:
			}
		}
	}

	return &staticRegisteredService{r, id, member}, nil
}

func (s *StaticRegistry) Close() error {
	return nil
}
