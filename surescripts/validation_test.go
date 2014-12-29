package surescripts

import (
	"testing"

	"github.com/sprucehealth/backend/encoding"
)

func TestAgeCalculation(t *testing.T) {
	dob := encoding.DOB{
		Year:  2014,
		Month: 1,
		Day:   1,
	}

	if is18YearsOfAge(dob) {
		t.Fatal("Expected the age to be < 18 years")
	}

	dob.Year = 1995
	dob.Month = 1
	dob.Day = 1

	if !is18YearsOfAge(dob) {
		t.Fatal("Expected the age to be > 18 years")
	}

}
