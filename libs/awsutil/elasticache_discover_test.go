package awsutil

import (
	"os"
	"testing"
	"time"
)

func TestElastiCacheDiscover(t *testing.T) {
	host := os.Getenv("TEST_ELASTICACHE_DISCOVERY_HOST")
	if host == "" {
		t.Skip("TEST_ELASTICACHE_DISCOVERY_HOST is not defined")
	}
	hosts, ver, err := ElastiCacheDiscover(host, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", hosts)
	if _, _, err := ElastiCacheDiscover(host, ver); err != ErrNoChange {
		t.Fatal("Expected ErrNoChange")
	}
}

func TestElastiCacheDiscoverer(t *testing.T) {
	host := os.Getenv("TEST_ELASTICACHE_DISCOVERY_HOST")
	if host == "" {
		t.Skip("TEST_ELASTICACHE_DISCOVERY_HOST is not defined")
	}
	d, err := NewElastiCacheDiscoverer(host, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", d.Hosts())
	d.Stop()
}
