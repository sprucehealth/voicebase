package grpcdns

import (
	"sync"
	"testing"
	"time"

	"github.com/sprucehealth/backend/libs/test"
	"google.golang.org/grpc"
	"google.golang.org/grpc/naming"
)

type testLookuper struct {
	mu    sync.Mutex
	addrs []string
}

func (l *testLookuper) lookup() ([]string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.addrs, nil
}

func (l *testLookuper) setAddrs(a []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.addrs = a
}

func TestHostPortLookuperFromTarget(t *testing.T) {
	l, err := lookuperFromTarget("sprucehealth.com:80")
	test.OK(t, err)
	ll, ok := l.(*hostPortLookuper)
	test.Assert(t, ok, "Expected type *hostPortLookuper got %T", l)
	test.Equals(t, "sprucehealth.com", ll.host)
	test.Equals(t, 80, ll.port)
}

func TestSRVLookuperFromTarget(t *testing.T) {
	l, err := lookuperFromTarget("_http._tcp.sprucehealth.com")
	test.OK(t, err)
	ll, ok := l.(*srvLookuper)
	test.Assert(t, ok, "Expected type *srvLookuper got %T", l)
	test.Equals(t, "http", ll.service)
	test.Equals(t, "tcp", ll.proto)
	test.Equals(t, "sprucehealth.com", ll.domain)
}

func TestSRVLookuper(t *testing.T) {
	// TODO: should have our own test domain for this to be more reliable
	l := &srvLookuper{
		service: "http",
		proto:   "tcp",
		domain:  "mxtoolbox.com",
	}
	addr, err := l.lookup()
	test.OK(t, err)
	test.Equals(t, []string{"mxtoolbox.com:80"}, addr)
}

func TestHostPortLookuper(t *testing.T) {
	l := &hostPortLookuper{
		host: "o4.sendgrid.sprucehealth.com",
		port: 1234,
	}
	addr, err := l.lookup()
	test.OK(t, err)
	test.Equals(t, []string{"167.89.97.19:1234"}, addr)
}

func TestWatcher(t *testing.T) {
	l := &testLookuper{
		addrs: []string{"one"},
	}
	w := &watcher{
		lookuper: l,
		target:   "xx",
		updateCh: make(chan []*naming.Update),
		addr:     make(map[string]struct{}),
	}

	// Initial list
	updates, err := w.update()
	test.OK(t, err)
	test.Equals(t, []*naming.Update{{Op: naming.Add, Addr: "one"}}, updates)

	// Should be no changes
	updates, err = w.update()
	test.OK(t, err)
	test.Equals(t, ([]*naming.Update)(nil), updates)

	// Additional host
	l.setAddrs([]string{"one", "two"})
	updates, err = w.update()
	test.OK(t, err)
	test.Equals(t, []*naming.Update{{Op: naming.Add, Addr: "two"}}, updates)

	// Remove host
	l.setAddrs([]string{"two"})
	updates, err = w.update()
	test.OK(t, err)
	test.Equals(t, []*naming.Update{{Op: naming.Delete, Addr: "one"}}, updates)

	// Add and remove host
	l.setAddrs([]string{"three"})
	updates, err = w.update()
	test.OK(t, err)
	test.Equals(t, []*naming.Update{
		{Op: naming.Add, Addr: "three"},
		{Op: naming.Delete, Addr: "two"},
	}, updates)

	// Empty host list shouldn't cause a change
	l.setAddrs([]string{})
	updates, err = w.update()
	test.AssertNotNil(t, err)
	test.AssertNil(t, updates)
}

func TestNonBlockingDial(t *testing.T) {
	l := &testLookuper{addrs: []string{}}
	r := &resolver{
		interval: time.Millisecond * 50,
		lookuper: l,
	}
	ch := make(chan interface{}, 1)
	go func() {
		conn, err := grpc.Dial("target", grpc.WithInsecure(), grpc.WithBalancer(grpc.RoundRobin(r)))
		if err != nil {
			ch <- err
		} else {
			ch <- conn
		}
	}()
	select {
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for Dial")
	case v := <-ch:
		if err, ok := v.(error); ok {
			t.Fatalf("Dial failed: %s", err)
		}
	}
}
