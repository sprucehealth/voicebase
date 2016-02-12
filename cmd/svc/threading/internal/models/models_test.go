package models

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestThreadID(t *testing.T) {
	t.Parallel()
	var id ThreadID
	id.Prefix = threadIDPrefix

	// Empty/invalid state marshaling
	b, err := id.MarshalText()
	test.OK(t, err)
	test.Equals(t, []byte(nil), b)
	test.Equals(t, "", id.String())

	// Valid unmarshaling
	id, err = ParseThreadID("t_00000000002D4")
	test.OK(t, err)
	test.Equals(t, uint64(1234), id.Val)
	test.Equals(t, true, id.IsValid)

	// Valid marshaling
	b, err = id.MarshalText()
	test.OK(t, err)
	test.Equals(t, []byte("t_00000000002D4"), b)
	test.Equals(t, "t_00000000002D4", id.String())
}
