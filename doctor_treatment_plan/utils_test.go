package doctor_treatment_plan

import "testing"

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
