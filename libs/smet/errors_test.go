package smet

import (
	"errors"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestError(t *testing.T) {
	Error("terror", errors.New("1"))
	Errorf("terror", "An error %d", 2)
	c := GetCounter("terror")
	test.Equals(t, uint64(2), c.Count())
}
