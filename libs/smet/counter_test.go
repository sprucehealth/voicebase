package smet

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestCounter(t *testing.T) {
	c := GetCounter("t1")
	c.Inc()
	c = GetCounter("t1")
	c.AddN(3)
	c2 := GetCounter("t2")
	c2.Inc()
	test.Equals(t, uint64(4), c.Count())
	test.Equals(t, uint64(1), c2.Count())
}
