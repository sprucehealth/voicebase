package hint

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
)

func TestTransformDOB(t *testing.T) {

	dob := "1981-01-31"
	date, err := transformDOB(dob)
	if err != nil {
		t.Fatalf(err.Error())
	}

	transformedDate := &sync.Patient_Date{
		Day:   31,
		Year:  1981,
		Month: 1,
	}

	if *date != *transformedDate {
		t.Fatalf("Date %s not transformed as expected %#v ", dob, date)
	}
}
