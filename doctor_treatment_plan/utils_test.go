package doctor_treatment_plan

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
)

func TestParseSections(t *testing.T) {
	cases := map[string]Sections{
		"":                   AllSections,
		"all":                AllSections,
		"note":               NoteSection,
		"regimen":            RegimenSection,
		"treatments":         TreatmentsSection,
		"scheduled_messages": ScheduledMessagesSection,
		"note,regimen":       NoteSection | RegimenSection,
		"all,note":           AllSections,

		"unknown":      NoSections,
		"unknown,note": NoteSection,
	}
	for str, sec := range cases {
		if s := parseSections(str); s != sec {
			t.Errorf(`parseSections("%s") = %d (%s). Expected %d`, str, s, s.String(), sec)
		}
	}
}

type mockDataAPI_validateTreatments struct {
	api.DataAPI
}

func (m *mockDataAPI_validateTreatments) SetDrugDescription(description *api.DrugDescription) error {
	return nil
}
func (m *mockDataAPI_validateTreatments) DrugDescriptions(queries []*api.DrugDescriptionQuery) ([]*api.DrugDescription, error) {
	return make([]*api.DrugDescription, len(queries)), nil
}

type erxAPI_validateTreatments struct {
	erx.ERxAPI
}

func (e *erxAPI_validateTreatments) SelectMedication(clinicianID int64, drugInternalName, dosageStrength string) (*erx.MedicationSelectResponse, error) {
	return &erx.MedicationSelectResponse{}, nil
}

func TestValidateDuplicateTreatments(t *testing.T) {

	treatments := []*common.Treatment{
		{
			DrugInternalName:    "Name (Form - Route)",
			DosageStrength:      "10%",
			DispenseValue:       10,
			DispenseUnitID:      encoding.DeprecatedNewObjectID(10),
			PatientInstructions: "aegihag",
			DrugDBIDs: map[string]string{
				"aegkhajg": "lakegh",
			},
		},
		{
			DrugInternalName:    "Name (Form - Route)",
			DosageStrength:      "10%",
			DispenseValue:       10,
			DispenseUnitID:      encoding.DeprecatedNewObjectID(10),
			PatientInstructions: "aegihag",
			DrugDBIDs: map[string]string{
				"aegkhajg": "lakegh",
			},
		},
	}

	if err := validateTreatments(treatments, nil, nil, 0); err == nil {
		t.Fatal("Expected error for duplicate treatments but got none.")
	}

	if err := validateTreatments(treatments[1:], &mockDataAPI_validateTreatments{}, &erxAPI_validateTreatments{}, 0); err != nil {
		t.Fatal(err)
	}

}
