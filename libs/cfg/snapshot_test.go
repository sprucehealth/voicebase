package cfg

import "testing"

func TestSnapshotMarshal(t *testing.T) {
	snap := &Snapshot{
		values: map[string]interface{}{
			"int64":   int64(123),
			"float64": float64(123.123),
		},
	}
	b, err := snap.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var snap2 Snapshot
	if err := snap2.UnmarshalJSON(b); err != nil {
		t.Fatal(err)
	}
	if v, ok := snap2.values["int64"].(int64); !ok {
		t.Fatalf("Expected int64, got %T", snap2.values["int64"])
	} else if e := snap.values["int64"].(int64); v != e {
		t.Fatalf("Expected %d, got %d", e, v)
	}
	if v, ok := snap2.values["float64"].(float64); !ok {
		t.Fatalf("Expected float64, got %T", snap2.values["float64"])
	} else if e := snap.values["float64"].(float64); v != e {
		t.Fatalf("Expected %f, got %f", e, v)
	}
}
