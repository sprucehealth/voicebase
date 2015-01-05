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

	// Test missing route
	m = &MedicationSelectResponse{
		GenericProductName:  "sulfacetamide sodium-sulfur 10%-5% liquid",
		RouteDescription:    "topical",
		DoseFormDescription: "liquid",
		StrengthDescription: "10%-5%",
	}
	if _, err := ParseGenericName(m); err == nil {
		t.Fatal("Expected an error")
	} else if err.Error() != "missing route" {
		t.Fatalf("Expected missing route, got '%s'", err.Error())
	}
}
