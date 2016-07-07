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
	"type": "q_type_autocomplete",
	"items": [{
		"text": "penicillAMINE",
		"answers" :{
			"43295": {
			"text": "5 times",
			"type" :"q_type_free_text"
			}
		}
	}, {
		"text": "Penicillin Benzath-Penicillin Proc"
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
	test.Equals(t, "5 times", ac.Answers[0].subAnswers()["43295"].(*freeTextAnswer).Text)
	test.Equals(t, "Penicillin Benzath-Penicillin Proc", ac.Answers[1].text())
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

func TestAutocomplete_transformForClient(t *testing.T) {
	expectedJSON := `{"type":"q_type_autocomplete","items":[{"text":"Hi","answers":{"11":{"type":"q_type_autocomplete","items":[{"text":"SubAnswerHello"}]}}}]}`

	acq := autocompleteAnswer{

		Answers: []topLevelAnswerItem{
			&answerItem{
				Text: "Hi",
				subScreens: []screen{
					&questionScreen{
						Questions: []question{
							&autocompleteQuestion{
								questionInfo: &questionInfo{
									ID: "11",
								},
								answer: &autocompleteAnswer{
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

	data, err := acq.transformForClient()
	if err != nil {
		t.Fatal(err)
	}

	jsonData, err := json.Marshal(data)
	test.OK(t, err)

	if expectedJSON != string(jsonData) {
		t.Fatalf("Expected `%s`, got `%s`", expectedJSON, string(jsonData))
	}
}

func TestAutocomplete_equals(t *testing.T) {
	acq := &autocompleteAnswer{
		Answers: []topLevelAnswerItem{
			&answerItem{
				Text: "Hi",
				SubAnswers: map[string]patientAnswer{
					"11": &autocompleteAnswer{
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
