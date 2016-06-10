package client

import (
	"testing"

	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/test"
)

func TestSectionWithTriageScreens(t *testing.T) {
	section := &saml.Section{
		Screens: []*saml.Screen{
			{
				Type: saml.ScreenTypeWarningPopup,
				Condition: &saml.Condition{
					Op:               "answer_equals",
					Question:         "question1",
					PotentialAnswers: []string{"answer1", "answer2"},
				},
			},
			{
				Type: saml.ScreenTypeTriage,
				Condition: &saml.Condition{
					Op:               "answer_equals",
					Question:         "question1",
					PotentialAnswers: []string{"answer1", "answer2"},
				},
			},
		},
	}

	layoutSection, err := transformSection(section)
	test.OK(t, err)
	test.Equals(t, 2, len(layoutSection.Screens))
	test.Equals(t, saml.ScreenTypeWarningPopup, layoutSection.Screens[0].Type)
	test.Equals(t, saml.ScreenTypeTriage, layoutSection.Screens[1].Type)
	test.Equals(t, &layout.Condition{
		Operation: "and",
		Operands: []*layout.Condition{
			{
				Operation:          "answer_equals",
				QuestionID:         "question1",
				PotentialAnswersID: []string{"answer1", "answer2"},
				Operands:           make([]*layout.Condition, 0),
			},
			{
				Operation:  "boolean_equals",
				BoolValue:  ptr.Bool(false),
				DataSource: "preference.optional_triage",
			},
		},
	}, layoutSection.Screens[0].Condition)
	test.Equals(t, &layout.Condition{
		Operation: "and",
		Operands: []*layout.Condition{
			{
				Operation:          "answer_equals",
				QuestionID:         "question1",
				PotentialAnswersID: []string{"answer1", "answer2"},
				Operands:           make([]*layout.Condition, 0),
			},
			{
				Operation:  "boolean_equals",
				BoolValue:  ptr.Bool(false),
				DataSource: "preference.optional_triage",
			},
		},
	}, layoutSection.Screens[1].Condition)
}
