package manager

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

var questionJSON = `
{
	"question": "q_current_medications_entry",
	"question_id": "40638",
	"question_title": "Which medications do you take or use regularly?",
	"question_title_has_tokens": false,
	"question_type": "q_type_autocomplete",
	"type": "q_type_autocomplete",
	"question_summary": "Medications taken or used",
	"additional_fields": {
		"add_button_text": "Add Medication",
		"add_text": "Add Medication",
		"empty_state_text": "No medications specified",
		"placeholder_text": "Type to add a medication",
		"remove_button_text": "Remove Medication",
		"save_button_text": "Save"
	},
	"condition": {
		"op": "answer_contains_any",
		"type": "answer_contains_any",
		"question": "q_regularly_taking_medications",
		"question_id": "40637",
		"potential_answers_id": ["126596"],
		"potential_answers": ["q_regularly_taking_medications_yes"]
	},
	"to_prefill": true,
	"prefilled_with_previous_answers": false,
	"required": true,
	"to_alert": false,
	"alert_text": "",
	"answers": [{
		"answer_id": "64457",
		"type": "q_type_autocomplete",
		"question_id": "40638",
		"answer_text": "penicillAMINE"
	}, {
		"answer_id": "64458",
		"type": "q_type_autocomplete",
		"question_id": "40638",
		"answer_text": "Penicillin Benzath-Penicillin Proc"
	}]
}
`

type mockDataSource_question struct {
	q question
	questionAnswerDataSource
}

func (m *mockDataSource_question) question(questionID string) question {
	return m.q
}

func (m *mockDataSource_question) valueForKey(key string) []byte {
	return nil
}

func TestQuestion_requirementsMet(t *testing.T) {
	s := &multipleChoiceQuestion{
		questionInfo: &questionInfo{
			Required: true,
		},
		answer: &multipleChoiceAnswer{},
	}

	// required question that has an empty answer set should not have its requirements met.
	if _, err := s.requirementsMet(&mockDataSource_question{}); err == nil {
		t.Fatal("Expected requirements to not be met for question")
	}

	// now lets make the answer non-empty
	s.answer.Answers = []topLevelAnswerItem{&multipleChoiceAnswerSelection{}, &multipleChoiceAnswerSelection{}}

	// optional question with  answer should have its requirements met
	s.Required = false
	if res, err := s.requirementsMet(&mockDataSource_question{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected requirements to be met for question")
	}

	// optional question with no answer should have its requriements met
	s.answer = nil
	if res, err := s.requirementsMet(&mockDataSource_question{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected requirements to be met for question")
	}

	// required question with no answer should _not_ have its requirements met
	s.Required = true
	if res, err := s.requirementsMet(&mockDataSource_question{}); err == nil || res {
		t.Fatal("Expected requirements to not be met if no answer for a required question")
	}

	// hidden question, even if required, should have its requirements met
	s.setVisibility(hidden)
	if res, err := s.requirementsMet(&mockDataSource_question{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected requirements to be met for question")
	}
}

func TestQuestion_staticCopy(t *testing.T) {
	q1 := &questionInfo{
		Title:          "Hi <parent_answer_text>",
		TitleHasTokens: true,
	}

	q2 := q1.staticInfoCopy(map[string]string{"answer": "spruce"}).(*questionInfo)
	test.Equals(t, "Hi spruce", q2.Title)
}
