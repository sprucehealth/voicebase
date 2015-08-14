package ratelimit

import (
	"os"
	"testing"

	"gopkgs.com/memcache.v2"
)

func TestMemcache(t *testing.T) {
	host := os.Getenv("TEST_MEMCACHED")
	if host == "" {
		t.Skip("TEST_MEMCACHED not set")
	}

	mc, err := memcache.New(host)
	if err != nil {
		t.Fatal(err)
	}

	rl := NewMemcache(mc, 5, 10)
	for i := 0; i < 5; i++ {
		if s, err := rl.Check("test", 1); err != nil {
			t.Fatal(err)
		} else if !s {
			t.Fatalf("Check did not succeed when it should have (iteration %d)", i)
		}
	}
	if s, err := rl.Check("test", 1); err != nil {
		t.Fatal(err)
	} else if s {
		t.Fatal("Check succeed when it should not have")
	}
}
