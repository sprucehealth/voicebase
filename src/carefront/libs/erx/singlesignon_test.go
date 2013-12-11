package erx

import (
	"testing"
)

func BenchmarkSingleSignonGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateSingleSignOn()
	}
}
