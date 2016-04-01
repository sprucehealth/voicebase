package model

import (
	"testing"

	"github.com/sprucehealth/backend/test"
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
