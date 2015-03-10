package treatment_plan

import (
	"testing"

	"github.com/sprucehealth/backend/common"
)

func TestFullTreatmentName(t *testing.T) {
	tc := []struct {
		t *common.Treatment
		n string
	}{
		{
			t: &common.Treatment{
				DrugName:       "Doxycycline Monohydrate",
				DosageStrength: "monohydrate 100 mg",
				DrugForm:       "capsule",
			},
			n: "Doxycycline monohydrate 100 mg capsule",
		},
		{
			t: &common.Treatment{
				DrugName:       "Doxycycline Monohydrate",
				DosageStrength: "100 mg",
				DrugForm:       "capsule",
			},
			n: "Doxycycline Monohydrate 100 mg capsule",
		},
	}
	for _, c := range tc {
		if n := fullTreatmentName(c.t); n != c.n {
			t.Errorf("fullTreatmentName(%+v) = '%s', expected '%s'", c.t, n, c.n)
		}
	}
}
