package dosespot

import (
	"encoding/xml"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestPharmacyUnmarshaler(t *testing.T) {
	var p Pharmacy
	err := xml.Unmarshal([]byte(`
		<pharmacy>
			<PharmacyId>123</PharmacyId>
			<PharmacySpecialties>this, that, other</PharmacySpecialties>
		</pharmacy>`), &p)
	test.OK(t, err)
	test.Equals(t, int64(123), p.PharmacyID)
	test.Equals(t, PharmacySpecialties([]string{"this", "that", "other"}), p.Specialties)
}
