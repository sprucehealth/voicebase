package manager

import (
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
	"github.com/sprucehealth/backend/libs/test"
)

func TestAutocomplete_AnswerUnmarshalFromMap(t *testing.T) {
	var clientJSON = `
{
	"answers": [{
		"answer_id": "64457",
		"question_id": "43332",
		"potential_answer_id": null,
		"answer_text": "penicillAMINE",
		"type": "q_type_autocomplete",
		"answers" :[{
			"answer_id": "64440",
			"answer_text": "5 times",
			"question_id": "43295",
			"type" :"q_type_free_text"
		}]
	}, {
		"answer_id": "64458",
		"question_id": "43332",
		"potential_answer_id": null,
		"answer_text": "Penicillin Benzath-Penicillin Proc",
		"type": "q_type_autocomplete"
	}]
}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(clientJSON), &data); err != nil {
		t.Fatal(err)
	}

	var ac autocompleteAnswer
	if err := ac.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 2, len(ac.Answers))
	test.Equals(t, "penicillAMINE", ac.Answers[0].text())
	test.Equals(t, 1, len(ac.Answers[0].subAnswers()))
	test.Equals(t, "5 times", ac.Answers[0].subAnswers()[0].(*freeTextAnswer).Text)
	test.Equals(t, "Penicillin Benzath-Penicillin Proc", ac.Answers[1].text())
	test.Equals(t, "43332", ac.QuestionID)
}

func TestAutocomplete_UnmarshalProtobuf(t *testing.T) {
	pb := &intake.AutocompletePatientAnswer{
		Answers: []string{
			"Hi",
			"Hello",
			"Howru",
		},
	}

	data, err := proto.Marshal(pb)
	if err != nil {
		t.Fatal(err)
	}

	var ac autocompleteAnswer
	if err := ac.unmarshalProtobuf(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 3, len(ac.Answers))
	test.Equals(t, "Hi", ac.Answers[0].text())
	test.Equals(t, "Hello", ac.Answers[1].text())
	test.Equals(t, "Howru", ac.Answers[2].text())
}

func TestAutocomplete_transformToProtobuf(t *testing.T) {
	acq := autocompleteAnswer{
		QuestionID: "10",
		Answers: []topLevelAnswerItem{
			&answerItem{
				Text: "Answer1",
			},
			&answerItem{
				Text: "Answer2",
			},
		},
	}

	pb, err := acq.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}

	acPb, ok := pb.(*intake.AutocompletePatientAnswer)
	if !ok {
		t.Fatalf("Expected *intake.AutocompletePatientAnswer but got %T", pb)
	}

	test.Equals(t, 2, len(acPb.Answers))
	test.Equals(t, "Answer1", acPb.Answers[0])
	test.Equals(t, "Answer2", acPb.Answers[1])
}

func TestAutocomplete_MarshalJSONForClient(t *testing.T) {
	expectedJSON := `{"question_id":"10","potential_answers":[{"answer_text":"Hi","answers":[{"question_id":"11","potential_answers":[{"answer_text":"SubAnswerHello"}]}]}]}`

	acq := autocompleteAnswer{
		QuestionID: "10",
		Answers: []topLevelAnswerItem{
			&answerItem{
				Text: "Hi",
				subScreens: []screen{
					&questionScreen{
						Questions: []question{
							&autocompleteQuestion{
								questionInfo: &questionInfo{},
								answer: &autocompleteAnswer{
									QuestionID: "11",
									Answers: []topLevelAnswerItem{
										&answerItem{
											Text: "SubAnswerHello",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	data, err := acq.marshalJSONForClient()
	if err != nil {
		t.Fatal(err)
	}

	if expectedJSON != string(data) {
		t.Fatalf("Expected `%s`, got `%s`", expectedJSON, string(data))
	}
}

func TestAutocomplete_equals(t *testing.T) {
	acq := &autocompleteAnswer{
		QuestionID: "10",
		Answers: []topLevelAnswerItem{
			&answerItem{
				Text: "Hi",
				SubAnswers: []patientAnswer{
					&autocompleteAnswer{
						QuestionID: "11",
						Answers: []topLevelAnswerItem{
							&answerItem{
								Text: "SubAnswerHello",
							},
						},
					},
				},
			},
		},
	}

	// answer should match itself
	if !acq.equals(acq) {
		t.Fatal("Answer expected to match itself")
	}

	// subanswers shouldn't play a role in equality given how client is setting the
	// answers (one question at a time)
	other := &autocompleteAnswer{
		QuestionID: "10",
		Answers: []topLevelAnswerItem{
			&answerItem{
				Text: "Hi",
			},
		},
	}

	if !acq.equals(other) {
		t.Fatal("Answer expected to match even when subanswers are different")
	}

	// answer with different items should not match
	acq = &autocompleteAnswer{
		QuestionID: "10",
		Answers: []topLevelAnswerItem{
			&answerItem{
				Text: "dagag",
			},
			&answerItem{
				Text: "Hagagi",
			},
		},
	}

	if acq.equals(other) {
		t.Fatal("Answer expected not to match")
	}

}
