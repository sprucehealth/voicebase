package cloudwatchlogs

import (
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	tm := Time{time.Unix(1, 22*1e6)}

	b, err := tm.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if s, want := string(b), "1022"; s != want {
		t.Fatalf("MarshalJSON(%+v) = '%s'. Expected '%s'", tm.Time, s, want)
	}
	var tm2 Time
	if err := tm2.UnmarshalJSON(b); err != nil {
		t.Fatal(err)
	}
	if tm != tm2 {
		t.Fatalf("%+v != %+v", tm, tm2)
	}
}
