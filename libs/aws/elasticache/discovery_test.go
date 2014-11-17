package elasticache

import (
	"os"
	"testing"
	"time"
)

func TestDiscover(t *testing.T) {
	host := os.Getenv("TEST_ELASTICACHE_DISCOVERY_HOST")
	if host == "" {
		t.Skip("TEST_ELASTICACHE_DISCOVERY_HOST is not defined")
	}
	hosts, ver, err := Discover(host, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", hosts)
	if _, _, err := Discover(host, ver); err != ErrNoChange {
		t.Fatal("Expected ErrNoChange")
	}
}

func TestDiscoverer(t *testing.T) {
	host := os.Getenv("TEST_ELASTICACHE_DISCOVERY_HOST")
	if host == "" {
		t.Skip("TEST_ELASTICACHE_DISCOVERY_HOST is not defined")
	}
	d, err := NewDiscoverer(host, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", d.Hosts())
	d.Stop()
}
