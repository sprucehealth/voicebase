package analytics

import "testing"

func TestIDGen(t *testing.T) {
	id, err := newID()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ID: %d\n", id)
}

func BenchmarkIDGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newID()
	}
	b.ReportAllocs()
}
