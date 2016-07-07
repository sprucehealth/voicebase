package manager

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

var autocompleteJSON = `
{
	"question": "q_current_medications_entry",
	"id": "40638",
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
	"alert_text": ""
}
`

func TestAutocomplete_Parsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(autocompleteJSON), &data); err != nil {
		t.Fatal(err)
	}

	acq := &autocompleteQuestion{}
	if err := acq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "Which medications do you take or use regularly?", acq.questionInfo.Title)
	test.Equals(t, "40638", acq.questionInfo.ID)
	test.Equals(t, "Add Medication", acq.AddButtonText)
	test.Equals(t, "Add Medication", acq.AddText)
	test.Equals(t, "Type to add a medication", acq.PlaceholderText)
	test.Equals(t, "Remove Medication", acq.RemoveButtonText)
	test.Equals(t, "Save", acq.SaveButtonText)
}

func TestAutocomplete_staticInfoCopy(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(autocompleteJSON), &data); err != nil {
		t.Fatal(err)
	}

	acq := &autocompleteQuestion{}
	if err := acq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	// nullify answers given that they are not static information
	acq.answer = nil

	acq2 := acq.staticInfoCopy(nil).(*autocompleteQuestion)
	test.Equals(t, acq, acq2)
}

func TestAutocomplete_Answer(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(autocompleteJSON), &data); err != nil {
		t.Fatal(err)
	}

	acq := &autocompleteQuestion{}
	if err := acq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	acqAnswer := &autocompleteAnswer{
		Answers: []topLevelAnswerItem{
			&answerItem{
				Text: "Hello",
			},
		},
	}

	if err := acq.setPatientAnswer(acqAnswer); err != nil {
		t.Fatal(err)
	}

	// allow empty answers
	acqAnswer = &autocompleteAnswer{}

	if err := acq.setPatientAnswer(acqAnswer); err != nil {
		t.Fatal(err)
	}

	acqAnswer = &autocompleteAnswer{
		Answers: []topLevelAnswerItem{
			&answerItem{},
			&answerItem{
				Text: "Hello",
			},
		},
	}

	if err := acq.setPatientAnswer(acqAnswer); err == nil {
		t.Fatal("Expected invalid answer but got valid answer")
	}
}

func TestAutocomplete_requirementsMet_subscreens(t *testing.T) {
	subQ := &freeTextQuestion{
		questionInfo: &questionInfo{
			Required: true,
		},
	}

	// required question, with answers having subscreens, should have its requirements met only if
	// all the screens requirements are met.
	s := &autocompleteQuestion{
		questionInfo: &questionInfo{
			Required: true,
		},
		answer: &autocompleteAnswer{
			Answers: []topLevelAnswerItem{
				&answerItem{
					Text: "agkahg",
					subScreens: []screen{
						&questionScreen{
							screenInfo: &screenInfo{},
							Questions: []question{
								subQ,
								&freeTextQuestion{
									questionInfo: &questionInfo{
										Required: true,
									},
									answer: &freeTextAnswer{
										Text: "Agag",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if _, err := s.requirementsMet(&mockDataSource_question{}); err == nil {
		t.Fatal("Expected requirements to not be met for question")
	} else if err != errSubQuestionRequirements {
		t.Fatalf("Expected subquestions requirements error but got %T instead", err)
	}

	// now lets make all required subquestions have an answer
	// and all requirements should be met.
	subQ.answer = &freeTextAnswer{Text: "adgkb"}
	if res, err := s.requirementsMet(&mockDataSource_question{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected requirements to be met for question")
	}

	// lets have a question with subscreens that are all just optional
	// and screens that are not question screens and ensure
	// that the requirements should be met
	s = &autocompleteQuestion{
		questionInfo: &questionInfo{
			Required: true,
		},
		answer: &autocompleteAnswer{
			Answers: []topLevelAnswerItem{
				&answerItem{
					Text: "agkahg",
					subScreens: []screen{
						&questionScreen{
							screenInfo: &screenInfo{},
							Questions: []question{
								&freeTextQuestion{
									questionInfo: &questionInfo{},
								},
							},
						},
						&warningPopupScreen{
							screenInfo: &screenInfo{
								screenClientData: &screenClientData{},
							},
						},
					},
				},
			},
		},
	}
	if res, err := s.requirementsMet(&mockDataSource_question{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected requirements to be met for question")
	}
}
