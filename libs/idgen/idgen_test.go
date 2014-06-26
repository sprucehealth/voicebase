package idgen

import "testing"

func TestIDGen(t *testing.T) {
	id, err := NewID()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ID: %d\n", id)
}

func BenchmarkIDGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewID()
	}
	b.ReportAllocs()
}
