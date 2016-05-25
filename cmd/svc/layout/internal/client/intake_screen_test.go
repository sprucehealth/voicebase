package client

import (
	"testing"

	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/test"
)

func TestTriageVisitScreen(t *testing.T) {
	triageScreen := &saml.Screen{
		Type: "screen_type_triage",
	}

	transformedScreen, err := transformScreen(triageScreen)
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, &layout.Condition{
		Operation:  "boolean_equals",
		BoolValue:  ptr.Bool(false),
		DataSource: "preference.optional_triage",
	}, transformedScreen.Condition)

	triageScreen = &saml.Screen{
		Type: "screen_type_triage",
		Condition: &saml.Condition{
			Op:               "answer_equals",
			Question:         "question1",
			PotentialAnswers: []string{"answer1", "answer2"},
		},
	}

	transformedScreen, err = transformScreen(triageScreen)
	if err != nil {
		t.Fatal(err)
	}

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
	}, transformedScreen.Condition)
}
