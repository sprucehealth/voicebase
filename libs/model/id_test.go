package model

import (
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/test"
)

func TestObjectID(t *testing.T) {
	id := ObjectID{Prefix: "t_"}

	// Empty/invalid state marshaling
	b, err := id.MarshalText()
	test.OK(t, err)
	test.Equals(t, []byte(nil), b)
	test.Equals(t, "", id.String())

	// Valid unmarshaling
	test.OK(t, id.UnmarshalText([]byte("t_00000000002D4")))
	test.Equals(t, uint64(1234), id.Val)
	test.Equals(t, true, id.IsValid)

	// Valid marshaling
	b, err = id.MarshalText()
	test.OK(t, err)
	test.Equals(t, []byte("t_00000000002D4"), b)
	test.Equals(t, "t_00000000002D4", id.String())

	// Bad prefix
	test.Assert(t, id.UnmarshalText([]byte("x_00000000002D4")) != nil, "Expected error when parsing ID with bad prefix")
}

func TestObjectIDScan(t *testing.T) {
	cases := []struct {
		v   interface{}
		exp ObjectID
		err error
	}{
		{nil, ObjectID{}, nil},
		{uint64(123), ObjectID{IsValid: true, Val: 123}, nil},
		{int64(1234), ObjectID{IsValid: true, Val: 1234}, nil},
		{[]byte("222"), ObjectID{IsValid: true, Val: 222}, nil},
		{"333", ObjectID{IsValid: true, Val: 333}, nil},
		{true, ObjectID{}, errors.New("unsupported type for ObjectID.Scan: bool")},
		{[]byte("abc"), ObjectID{}, errors.New(`failed to scan ObjectID string 'abc'`)},
		{"abc", ObjectID{}, errors.New(`failed to scan ObjectID string 'abc'`)},
	}
	for _, c := range cases {
		var id ObjectID
		err := id.Scan(c.v)
		if c.err != nil {
			if err == nil {
				t.Errorf("Expected error scanning %#v", c.v)
			} else if c.err.Error() != errors.Cause(err).Error() {
				t.Errorf("Expected error '%s' when scanning %#v, got '%s'", c.err, c.v, errors.Cause(err))
			}
		} else if err != nil {
			t.Errorf("Unexpected error '%s' scanning %#v", err, c.v)
		}
		test.EqualsCase(t, fmt.Sprintf("%#v", c.v), c.exp, id)
	}
}

func TestObjectIDValue(t *testing.T) {
	cases := []struct {
		id ObjectID
		v  driver.Value
	}{
		{ObjectID{}, nil},
		{ObjectID{IsValid: false, Val: 123}, nil},
		{ObjectID{IsValid: true, Val: 0}, int64(0)},
		{ObjectID{IsValid: true, Val: 123}, int64(123)},
	}
	for _, c := range cases {
		v, err := c.id.Value()
		test.OK(t, err)
		test.EqualsCase(t, fmt.Sprintf("%#v", c.id), c.v, v)
	}
}

func BenchmarkObjectIDMarshalText(b *testing.B) {
	id := ObjectID{Prefix: "t_"}
	test.OK(b, id.UnmarshalText([]byte("t_00000000002D4")))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := id.MarshalText(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkObjectIDUnmarshalText(b *testing.B) {
	idString := []byte("t_00000000002D4")
	id := ObjectID{Prefix: "t_"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := id.UnmarshalText(idString); err != nil {
			b.Fatal(err)
		}
	}
}
