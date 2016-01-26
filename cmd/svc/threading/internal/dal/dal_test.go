package dal

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/test"
)

func TestTimeCursor(t *testing.T) {
	tm := time.Unix(1, 234567890)

	// sanity check mainly for documentation purposes
	test.Equals(t, int64(1234567890), tm.UnixNano())

	// should return the time in microsecond
	ms := formatTimeCursor(tm)
	test.Equals(t, "1234567", ms)

	tm2, err := parseTimeCursor(ms)
	test.OK(t, err)
	test.Equals(t, int64(1234567000), tm2.UnixNano())
}
