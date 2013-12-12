package erx

import (
	"os"
	"testing"
)

func BenchmarkSingleSignonGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		clinicKey := os.Getenv("DOSESPOT_CLINIC_KEY")
		userId := os.Getenv("DOSESPOT_USER_ID")
		clinicId := os.Getenv("DOSESPOT_CLINIC_ID")
		generateSingleSignOn(clinicKey, userId, clinicId)
	}
}
