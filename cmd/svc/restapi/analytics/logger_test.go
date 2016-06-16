package analytics

import (
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	t1 := Time(time.Unix(123123123, 0))
	b, err := t1.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	var t2 Time
	if err := t2.UnmarshalText(b); err != nil {
		t.Fatal(err)
	}
	if time.Time(t1).Unix() != time.Time(t2).Unix() {
		t.Fatalf("Time doesn't match: %+v != %+v", t1, t2)
	}
}
