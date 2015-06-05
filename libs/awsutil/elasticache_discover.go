package awsutil

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrNoChange = errors.New("elasticache: no change in host list")

type ElastiCacheDiscoverer struct {
	dhost       string
	ver         string
	hosts       []string
	subscribers map[chan<- []string]bool
	discCh      chan bool
	stopCh      chan bool
	mu          sync.RWMutex
	smu         sync.RWMutex
}

func NewElastiCacheDiscoverer(discoveryHost string, updateInterval time.Duration) (*ElastiCacheDiscoverer, error) {
	if strings.IndexByte(discoveryHost, ':') < 0 {
		discoveryHost += ":11211"
	}
	hosts, ver, err := ElastiCacheDiscover(discoveryHost, "")
	if err != nil {
		return nil, err
	}
	d := &ElastiCacheDiscoverer{
		dhost:       discoveryHost,
		ver:         ver,
		hosts:       hosts,
		subscribers: make(map[chan<- []string]bool),
		discCh:      make(chan bool, 1),
		stopCh:      make(chan bool),
	}
	go d.loop(updateInterval)
	return d, nil
}

// Update forces updating the host list
func (d *ElastiCacheDiscoverer) Update() {
	// Ignore a full channel since that means an update was already signalled
	select {
	case d.discCh <- true:
	default:
	}
}

func (d *ElastiCacheDiscoverer) Subscribe(ch chan<- []string) {
	d.smu.Lock()
	d.subscribers[ch] = true
	d.smu.Unlock()
}

func (d *ElastiCacheDiscoverer) Unsubscribe(ch chan<- []string) {
	d.smu.Lock()
	delete(d.subscribers, ch)
	d.smu.Unlock()
}

func (d *ElastiCacheDiscoverer) Hosts() []string {
	d.mu.RLock()
	hosts := d.hosts
	d.mu.RUnlock()
	return hosts
}

func (d *ElastiCacheDiscoverer) Stop() {
	close(d.stopCh)
}

func (d *ElastiCacheDiscoverer) loop(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-d.discCh:
		case <-t.C:
		}

		hosts, ver, err := ElastiCacheDiscover(d.dhost, d.ver)
		if err == ErrNoChange {
			continue
		} else if err != nil {
			log.Printf("[ERR] failed to discover memcached hosts: %s", err.Error())
			continue
		}

		d.ver = ver

		d.mu.Lock()
		d.hosts = hosts
		d.mu.Unlock()

		d.smu.RLock()
		for ch := range d.subscribers {
			// Ignore full channels
			select {
			case ch <- hosts:
			default:
			}
		}
		d.smu.RUnlock()
	}
}

// ElastiCacheDiscover returns the list of memcached nodes. prevVer is the
// version return by a previous call to Discovery and if there
// has been no change in nodes since the last call an error of
// ErrNoChange will be returned.
func ElastiCacheDiscover(host string, prevVer string) ([]string, string, error) {
	if strings.IndexByte(host, ':') < 0 {
		host += ":11211"
	}
	c, err := net.Dial("tcp", host)
	if err != nil {
		return nil, prevVer, err
	}
	tp := textproto.NewConn(c)
	defer tp.Close()
	if err := tp.Writer.PrintfLine("config get cluster"); err != nil {
		return nil, prevVer, err
	}
	// Skip the first response line since output is always 2 lines according to AWS docs.
	// CONFIG cluster 0 <size>
	if _, err := tp.ReadLine(); err != nil {
		return nil, prevVer, err
	}
	// Version number (incremented when host list changes)
	ver, err := tp.ReadLine()
	if err != nil {
		return nil, prevVer, err
	}
	ver = strings.TrimSpace(ver)
	if ver == prevVer {
		// Host list unchanged
		return nil, ver, ErrNoChange
	}
	// hostname|ip-address|port hostname|ip-address|port hostname|ip-address|port\r\n
	line, err := tp.ReadLine()
	if err != nil {
		return nil, prevVer, err
	}
	line = strings.TrimSpace(line)
	hs := strings.Split(line, " ")
	hosts := make([]string, len(hs))
	for i, h := range hs {
		p := strings.Split(h, "|")

		// Sanity check that we can parse the port
		if _, err := strconv.Atoi(p[2]); err != nil {
			return nil, prevVer, fmt.Errorf("elasticache: failed to parse port '%s'", p[2])
		}

		// AWS claims the IP may be empty but hostname is always set so try IP first
		// but fallback to hostname
		addr := p[1] + ":" + p[2]
		if p[1] == "" {
			addr = p[0] + ":" + p[2]
		}
		hosts[i] = addr
	}
	return hosts, ver, nil
}
