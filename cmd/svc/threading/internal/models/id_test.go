package models

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestObjectID(t *testing.T) {
	id := objectID{prefix: "t_"}

	// Empty/invalid state marshaling
	b, err := id.MarshalText()
	test.OK(t, err)
	test.Equals(t, []byte(nil), b)
	test.Equals(t, "", id.String())

	// Valid unmarshaling
	test.OK(t, id.UnmarshalText([]byte("t_00000000002D4")))
	test.Equals(t, uint64(1234), id.value)
	test.Equals(t, true, id.isValid)

	// Valid marshaling
	b, err = id.MarshalText()
	test.OK(t, err)
	test.Equals(t, []byte("t_00000000002D4"), b)
	test.Equals(t, "t_00000000002D4", id.String())
}
