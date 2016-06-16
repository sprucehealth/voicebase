package mcutil

import (
	"math"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestHRW(t *testing.T) {
	hrw := NewHRWServer([]string{"a", "b"})
	addrs, err := hrw.Servers()
	test.OK(t, err)
	test.Equals(t, 2, len(addrs))
	test.Equals(t, "tcp", addrs[0].Network())
	test.Equals(t, "a", addrs[0].String())

	counts := make(map[string]int)
	for i := 0; i < 10000; i++ {
		addr, err := hrw.PickServer(strconv.Itoa(i))
		test.OK(t, err)
		counts[addr.String()]++
	}
	if e := math.Abs(1.0 - float64(counts["a"])/float64(counts["b"])); e > 0.05 {
		t.Fatalf("Expected ~50/50 distribution, diff %f", e)
	}
}

func BenchmarkHRW(b *testing.B) {
	hrw := NewHRWServer([]string{"a", "b", "c", "d"})
	keys := make([]string, 1000)
	for i := 0; i < len(keys); i++ {
		keys[i] = strconv.Itoa(i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hrw.PickServer(keys[i%len(keys)])
	}
}
