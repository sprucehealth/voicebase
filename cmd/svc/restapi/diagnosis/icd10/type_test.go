package icd10

import "testing"

func TestDiagnosisKey(t *testing.T) {
	checkKey(Code("L71.0"), "diag_l710", t)
	checkKey(Code("L71"), "diag_l71", t)
	checkKey(Code("L71.1234"), "diag_l711234", t)
	checkKey(Code("L71.1XX4"), "diag_l711xx4", t)
}

func checkKey(code Code, expectedKey string, t *testing.T) {
	if code.Key() != expectedKey {
		t.Fatalf("Expected %s, got %s", expectedKey, code.Key())
	}
}
