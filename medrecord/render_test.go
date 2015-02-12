package medrecord

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/sprucehealth/backend/common"
)

func TestRender(t *testing.T) {
	ctx := &templateContext{
		Patient: &common.Patient{},
		PCP:     &common.PCP{},
		EmergencyContacts: []*common.EmergencyContact{
			&common.EmergencyContact{},
		},
		Agreements: map[string]time.Time{
			"privacy": time.Now(),
		},
	}

	cas := &caseContext{
		Case: &common.PatientCase{},
		Visits: []*visitContext{
			&visitContext{
				Visit: &common.PatientVisit{},
			},
		},
		Messages: []*caseMessage{
			&caseMessage{
				Media: []*caseMedia{
					&caseMedia{
						Type: "photo",
						URL:  "http://127.0.0.1/",
					},
					&caseMedia{
						Type: "audio",
						URL:  "http://127.0.0.1/",
					},
				},
			},
		},
	}
	ctx.Cases = append(ctx.Cases, cas)

	if err := mrTemplate.Execute(ioutil.Discard, ctx); err != nil {
		t.Fatal(err)
	}
}
