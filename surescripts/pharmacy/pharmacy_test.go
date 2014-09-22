package pharmacy

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestPharmacyStoreName(t *testing.T) {

	storeName := "Walgreens Pharmacy #6029"
	parsedName := removeStoreNumbersFromName(storeName)
	test.Equals(t, "Walgreens Pharmacy", parsedName)

	storeName = "Walgreens Pharmacy 6029"
	parsedName = removeStoreNumbersFromName(storeName)
	test.Equals(t, "Walgreens Pharmacy", parsedName)

	storeName = "Walgreens 6029 Pharmacy"
	parsedName = removeStoreNumbersFromName(storeName)
	test.Equals(t, storeName, parsedName)

	storeName = "Walgreens Pharmacy 2039a"
	parsedName = removeStoreNumbersFromName(storeName)
	test.Equals(t, "Walgreens Pharmacy", parsedName)

	storeName = "Walgreens Pharmacy a2039"
	parsedName = removeStoreNumbersFromName(storeName)
	test.Equals(t, storeName, parsedName)

	storeName = "Walgreens"
	parsedName = removeStoreNumbersFromName(storeName)
	test.Equals(t, "Walgreens", parsedName)

}

func BenchmarkPharmacyNameWithoutNumbers(b *testing.B) {
	for i := 0; i < b.N; i++ {
		removeStoreNumbersFromName("Walgreens Pharmacy #629")
	}
	b.ReportAllocs()
}
