package patient_case

import "testing"

func TestSpelledNumber(t *testing.T) {
	testSpelledNumber(t, 2, " two ")
	testSpelledNumber(t, 3, " three ")
	testSpelledNumber(t, 4, " four ")
	testSpelledNumber(t, 5, " five ")
	testSpelledNumber(t, 6, " six ")
	testSpelledNumber(t, 7, " seven ")
	testSpelledNumber(t, 8, " eight ")
	testSpelledNumber(t, 9, " nine ")
	testSpelledNumber(t, 10, " ten ")
	testSpelledNumber(t, 11, "")
}

func testSpelledNumber(t *testing.T, num int, expected string) {
	if expected != spellNumber(num) {
		t.Fatalf("Expected %s but got %s", expected, spellNumber(num))
	}
}
