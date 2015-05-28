package erx

import "testing"

func TestParseGenericName(t *testing.T) {
	m := &MedicationSelectResponse{
		GenericProductName:  "sulfacetamide sodium-sulfur 10%-5% topical liquid",
		RouteDescription:    "topical",
		DoseFormDescription: "liquid",
		StrengthDescription: "10%-5%",
	}
	if name, err := ParseGenericName(m); err != nil {
		t.Fatalf("Failed to parse %+v: %s", m, err.Error())
	} else if e := "sulfacetamide sodium-sulfur"; name != e {
		t.Fatalf("Expected '%s', got '%s'", e, name)
	}

	m = &MedicationSelectResponse{
		GenericProductName:  "bimatoprost topical ophthalmic 0.03% solution",
		RouteDescription:    "topical",
		DoseFormDescription: "solution",
		StrengthDescription: "0.03%",
	}
	if name, err := ParseGenericName(m); err != nil {
		t.Fatalf("Failed to parse %+v: %s", m, err.Error())
	} else if e := "bimatoprost"; name != e {
		t.Fatalf("Expected '%s', got '%s'", e, name)
	}
}
