package mcutil

import (
	"errors"
	"hash"
	"hash/fnv"
	"sync"

	"github.com/rainycape/memcache"
	"github.com/sprucehealth/backend/libs/awsutil"
)

var fnvPool = sync.Pool{
	New: func() interface{} {
		return fnv.New32a()
	},
}

type tcpAddr string

func (a tcpAddr) Network() string {
	return "tcp"
}

func (a tcpAddr) String() string {
	return string(a)
}

// HRWServers implements the highest random weight (aka rendezvous) hashing algorithm for picking cache servers.
type HRWServers struct {
	hosts   []*memcache.Addr
	hostMap map[int32]*memcache.Addr
	mu      sync.RWMutex
}

// NewHRWServer returns a new HRWServers with the provided host list.
func NewHRWServer(hosts []string) *HRWServers {
	hs := &HRWServers{}
	hs.SetHosts(hosts)
	return hs
}

// PickServer selects one server from the ones by managed by the Servers
// instance, based on the given key using Highest Random Weight
// (aka Rendezvous hashing).
// http://www.eecs.umich.edu/techreports/cse/96/CSE-TR-316-96.pdf
func (hs *HRWServers) PickServer(key string) (*memcache.Addr, error) {
	hs.mu.RLock()
	hm := hs.hostMap
	hs.mu.RUnlock()
	if len(hm) == 0 {
		return nil, errors.New("mcutil: no memcached hosts")
	}

	h := fnvPool.Get().(hash.Hash32)
	h.Write([]byte(key))
	d := int32(h.Sum32())
	h.Reset()
	fnvPool.Put(h)

	var max int
	var addr *memcache.Addr
	for ai, a := range hm {
		w := weight(ai, d)
		if addr == nil || w > max || (w == max && a.String() > addr.String()) {
			max = w
			addr = a
		}
	}

	return addr, nil
}

// Servers returns all the servers available.
func (hs *HRWServers) Servers() ([]*memcache.Addr, error) {
	hs.mu.RLock()
	hosts := hs.hosts
	hs.mu.RUnlock()
	return hosts, nil
}

// SetHosts updates the list of cache hosts.
func (hs *HRWServers) SetHosts(hosts []string) {
	addrs := hostsToMCAddr(hosts)
	hostMap := make(map[int32]*memcache.Addr, len(addrs))
	h := fnvPool.Get().(hash.Hash32)
	for _, a := range addrs {
		h.Write([]byte(a.String()))
		hostMap[int32(h.Sum32())] = a
		h.Reset()
	}
	fnvPool.Put(h)
	hs.mu.Lock()
	hs.hosts = addrs
	hs.hostMap = hostMap
	hs.mu.Unlock()
}

// ElastiCacheServers implements host discovery for ElastiCache. It uses HRW to pick servers.
type ElastiCacheServers struct {
	*HRWServers
	d      *awsutil.ElastiCacheDiscoverer
	ch     chan []string
	stopCh chan bool
}

// NewElastiCacheServers returns a new ElasticCacheServers picker using the provided discoverer.
func NewElastiCacheServers(d *awsutil.ElastiCacheDiscoverer) *ElastiCacheServers {
	ecs := &ElastiCacheServers{
		HRWServers: NewHRWServer(d.Hosts()),
		d:          d,
		ch:         make(chan []string, 1),
		stopCh:     make(chan bool),
	}
	d.Subscribe(ecs.ch)
	go ecs.loop()
	return ecs
}

// Stop terminates the server discovery
func (ecs *ElastiCacheServers) Stop() {
	close(ecs.stopCh)
	ecs.d.Stop()
}

// Update forces an update of cache servers
func (ecs *ElastiCacheServers) Update() {
	ecs.d.Update()
}

func (ecs *ElastiCacheServers) loop() {
	for {
		select {
		case hosts := <-ecs.ch:
			ecs.SetHosts(hosts)
		case <-ecs.stopCh:
			return
		}
	}
}

func hostsToMCAddr(hosts []string) []*memcache.Addr {
	addrs := make([]*memcache.Addr, len(hosts))
	for i, h := range hosts {
		addrs[i] = memcache.NewAddr(tcpAddr(h))
	}
	return addrs
}

func weight(s, d int32) int {
	v := (a * ((a*s + c) ^ d + c))
	if v < 0 {
		v += m
	}
	return int(v)
}

const (
	a = 1103515245    // multiplier
	c = 12345         // increment
	m = (1 << 31) - 1 // modulus (2**32-1)
)
