package cfg

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/hashicorp/consul/api"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func newConsulClient(t *testing.T) *api.Client {
	addr := os.Getenv("TEST_CONSUL_ADDRESS")
	if addr == "" {
		t.Skip("TEST_CONSUL_ADDRESS not set")
	}
	cli, err := api.NewClient(&api.Config{
		Address: addr,
	})
	if err != nil {
		t.Fatal(err)
	}
	return cli
}

func TestConsulStore(t *testing.T) {
	cli := newConsulClient(t)
	key := fmt.Sprintf("test/cfg/%d", rand.Int())
	store := newConsulStore(cli, key, metrics.NewRegistry())
	store.testCh = make(chan Snapshot, 64)
	if err := store.start(); err != nil {
		t.Fatal(err)
	}
	defer cli.KV().Delete(key, nil)
	defer store.Close()
	store.Register(&ValueDef{Name: "someint", Type: ValueTypeInt, Default: 0})
	store.Register(&ValueDef{Name: "somefloat", Type: ValueTypeFloat, Default: 0.0})
	store.Register(&ValueDef{Name: "somestring", Type: ValueTypeString, Default: ""})
	store.Register(&ValueDef{Name: "someduration", Type: ValueTypeDuration, Default: time.Duration(0)})
	snap := store.Snapshot()
	if snap.Len() != 0 {
		t.Fatal("Expected an empty snapshot on start")
	}

	if err := store.Update(map[string]interface{}{
		"someint":      123,
		"someduration": time.Hour,
	}); err != nil {
		t.Fatal(err)
	}
	snap = store.Snapshot()
	if snap.Len() != 2 {
		t.Fatalf("Expected a snapshot with 2 values instead of %d", snap.Len())
	}
	if v := snap.Int("someint"); v != 123 {
		t.Fatalf("Expected 123, got %d", v)
	}
	if v := snap.Duration("someduration"); v != time.Hour {
		t.Fatalf("Expected %d, got %d", time.Hour, v)
	}

	if err := store.Update(map[string]interface{}{
		"someint": 555,
	}); err != nil {
		t.Fatal(err)
	}
	snap = store.Snapshot()
	if snap.Len() != 2 {
		t.Fatalf("Expected a snapshot with 2 values instead of %d", snap.Len())
	}
	if v := snap.Int("someint"); v != 555 {
		t.Fatalf("Expected 555, got %d", v)
	}

	for i := 0; i < 3; i++ {
		select {
		case <-store.testCh:
		case <-time.After(time.Millisecond * 100):
			t.Fatalf("Expected more than %d values", i)
		}
	}

	// Make sure changes propagate across connections

	store2, err := NewConsulStore(cli, key, metrics.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()
	store2.Register(&ValueDef{Name: "someint", Type: ValueTypeInt, Default: 0})
	store2.Register(&ValueDef{Name: "somefloat", Type: ValueTypeFloat, Default: 0.0})
	store2.Register(&ValueDef{Name: "somestring", Type: ValueTypeString, Default: ""})
	store2.Register(&ValueDef{Name: "someduration", Type: ValueTypeDuration, Default: time.Duration(0)})

	if err := store2.Update(map[string]interface{}{
		"someint":      999,
		"someduration": time.Minute,
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case <-store.testCh:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("Timeout")
	}

	snap = store.Snapshot()
	if snap.Len() != 2 {
		t.Fatalf("Expected a snapshot with 2 values instead of %d", snap.Len())
	}
	if v := snap.Int("someint"); v != 999 {
		t.Fatalf("Expected 999, got %d", v)
	}
	if v := snap.Duration("someduration"); v != time.Minute {
		t.Fatalf("Expected %d, got %d", time.Minute, v)
	}
}
