package main

import (
	"errors"
	"hash/fnv"
	"sync"

	"github.com/sprucehealth/backend/libs/aws/elasticache"
	"github.com/sprucehealth/backend/third_party/gopkgs.com/memcache.v2"
)

type tcpAddr string

func (a tcpAddr) Network() string {
	return "tcp"
}

func (a tcpAddr) String() string {
	return string(a)
}

type HRWServers struct {
	hosts   []*memcache.Addr
	hostMap map[int32]*memcache.Addr
	mu      sync.RWMutex
}

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
		return nil, errors.New("no memcached hosts")
	}

	h := fnv.New32a()
	h.Write([]byte(key))
	d := int32(h.Sum32())

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

func (hs *HRWServers) SetHosts(hosts []string) {
	addrs := hostsToMCAddr(hosts)
	hostMap := make(map[int32]*memcache.Addr, len(addrs))
	for _, a := range addrs {
		h := fnv.New32a()
		h.Write([]byte(a.String()))
		hostMap[int32(h.Sum32())] = a
	}
	hs.mu.Lock()
	hs.hosts = addrs
	hs.hostMap = hostMap
	hs.mu.Unlock()
}

type ElastiCacheServers struct {
	*HRWServers
	d      *elasticache.Discoverer
	ch     chan []string
	stopCh chan bool
}

func NewElastiCacheServers(d *elasticache.Discoverer) *ElastiCacheServers {
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

func (ecs *ElastiCacheServers) Stop() {
	close(ecs.stopCh)
	ecs.d.Stop()
}

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
