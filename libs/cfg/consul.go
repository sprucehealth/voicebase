package cfg

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/hashicorp/consul/api"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	consulUpdateRetries    = 3
	consulUpdateRetryDelay = time.Millisecond * 50
	consulWaitTime         = time.Minute
)

type consulStore struct {
	*localStore
	cli              *api.Client
	key              string
	stopCh           chan struct{}
	waitTime         time.Duration
	testCh           chan Snapshot
	statUpdates      *metrics.Counter
	statConsulErrors *metrics.Counter
	updateMu         sync.Mutex // serializes updates
}

// NewConsulStore returns a Store that uses Consul as the backing storage for config
// values. Any instance of the Consul Store that uses the same cluster and key sees the
// same set of values regardless of which one does the Update. Values propogate using
// watches in the shared value so value should propagate quickly baring any breaks in
// the Consul cluster.
func NewConsulStore(cli *api.Client, key string, defs []*ValueDef, metricsRegistry metrics.Registry) (Store, error) {
	cs, err := newConsulStore(cli, key, defs, metricsRegistry)
	if err != nil {
		return nil, err
	}
	return cs, cs.start()
}

func newConsulStore(cli *api.Client, key string, defs []*ValueDef, metricsRegistry metrics.Registry) (*consulStore, error) {
	ls, err := NewLocalStore(defs)
	if err != nil {
		return nil, err
	}
	cs := &consulStore{
		localStore:       ls.(*localStore),
		cli:              cli,
		key:              key,
		stopCh:           make(chan struct{}),
		waitTime:         consulWaitTime,
		statUpdates:      metrics.NewCounter(),
		statConsulErrors: metrics.NewCounter(),
	}
	metricsRegistry.Add("updates", cs.statUpdates)
	metricsRegistry.Add("consul/error", cs.statConsulErrors)
	return cs, nil
}

func (cs *consulStore) start() error {
	values, modifyIndex, err := cs.fetchValues(true, 0)
	if err != nil {
		return err
	}
	if err := cs.localStore.update(values, false); err != nil {
		return err
	}
	go cs.loop(modifyIndex)
	return nil
}

func (cs *consulStore) Close() error {
	close(cs.stopCh)
	return nil
}

func (cs *consulStore) Update(update map[string]interface{}) error {
	cs.updateMu.Lock()
	defer cs.updateMu.Unlock()
	for i := 0; i < consulUpdateRetries; i++ {
		// Always fetch the current snapshot rather than relying on the
		// one that's already been pulled to avoid updating stale values.
		values, modifyIndex, err := cs.fetchValues(false, 0)
		if err != nil {
			return err
		}
		cs.localStore.values.Store(values)
		if err := cs.localStore.Update(update); err != nil {
			return err
		}
		newSnapshot := cs.localStore.Snapshot()
		b, err := newSnapshot.MarshalJSON()
		if err != nil {
			return err
		}
		kvp := &api.KVPair{
			Key:         cs.key,
			Value:       b,
			ModifyIndex: modifyIndex,
		}
		ok, _, err := cs.cli.KV().CAS(kvp, nil)
		if err != nil {
			cs.statConsulErrors.Inc(1)
			return fmt.Errorf("cfg.consul: failed to update values: %s", err)
		}
		if ok {
			cs.statUpdates.Inc(1)
			return nil
		}
		time.Sleep(consulUpdateRetryDelay)
	}
	return errors.New("cfg.consul: lost race to update")
}

func (cs *consulStore) fetchValues(allowStale bool, modifyIndex uint64) (map[string]interface{}, uint64, error) {
	opt := &api.QueryOptions{
		AllowStale: allowStale,
		WaitIndex:  modifyIndex,
		WaitTime:   cs.waitTime,
	}
	item, _, err := cs.cli.KV().Get(cs.key, opt)
	if err != nil {
		return nil, 0, err
	}
	if item == nil {
		// Sanity check
		snap := cs.Snapshot()
		if snap.Len() != 0 {
			// For now just log this as an error to surface it. If this is an issue that actually
			// happens then need to solve it (perhaps by ignoring the GET and retrying). For now
			// just want to see if it ever happens.
			golog.Errorf("cfg.consul: fetched empty item from consul but snapshot has values")
		}

		// Initialize an empty value
		_, _, err := cs.cli.KV().CAS(&api.KVPair{
			Key:         cs.key,
			Value:       []byte("{}"),
			ModifyIndex: 0,
		}, nil)
		return map[string]interface{}{}, 0, err
	}
	values, err := DecodeValues(item.Value)
	return values, item.ModifyIndex, err
}

func (cs *consulStore) loop(modifyIndex uint64) {
	for {
		select {
		case <-cs.stopCh:
			return
		default:
		}
		values, mi, err := cs.fetchValues(true, modifyIndex)
		if err != nil {
			cs.statConsulErrors.Inc(1)
			golog.Warningf("cfg.consul: failed to fetch values: %s", err)
			continue
		}
		if cs.testCh != nil {
			select {
			case cs.testCh <- Snapshot{values: values, defs: cs.localStore.defs}:
			default:
				panic("test channel overflow")
			}
		}
		modifyIndex = mi
		if err := cs.localStore.Update(values); err != nil {
			// This should never happen
			golog.Errorf("cfg.consul: failed to update: %s", err)
		}
	}
}
