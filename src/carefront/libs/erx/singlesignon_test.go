package erx

import (
	"os"
	"strconv"
	"testing"
)

func BenchmarkSingleSignonGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		clinicKey := os.Getenv("DOSESPOT_CLINIC_KEY")
		userId, _ := strconv.ParseInt(os.Getenv("DOSESPOT_USER_ID"), 10, 64)
		clinicId, _ := strconv.ParseInt(os.Getenv("DOSESPOT_CLINIC_ID"), 10, 64)
		generateSingleSignOn(clinicKey, userId, clinicId)
	}
}
