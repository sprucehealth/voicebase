package manager

import (
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
	"github.com/sprucehealth/backend/libs/test"
)

func TestMultipleChoice_unmarshalMapFromClient(t *testing.T) {
	clientJSON := `
	{
		"type": "q_type_multiple_choice",
		"potential_answers": [
			{
				"id": "143498",
				"text": "Benzoyl peroxide",
				"answers": {
					"43295": {
						"type" : "q_type_single_select",
						"potential_answer" : {
							"id": "143487"
						}
					},
					"43296": {
						"potential_answer" : {
							"id": "143489"
						},
						"type": "q_type_segmented_control"
					},
					"43297": {
						"text": "Not sure what strength it was.",
						"type" : "q_type_free_text"
					}
				}
			},
			{
				"id": "143499",
				"text" : "Testing"
			}
		]
	}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(clientJSON), &data); err != nil {
		t.Fatal(err)
	}

	var mca multipleChoiceAnswer
	if err := mca.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 2, len(mca.Answers))
	test.Equals(t, "143498", mca.Answers[0].potentialAnswerID())
	test.Equals(t, 3, len(mca.Answers[0].subAnswers()))
	test.Equals(t, "143487", mca.Answers[0].subAnswers()["43295"].(*singleSelectAnswer).Answer.potentialAnswerID())
	test.Equals(t, "143489", mca.Answers[0].subAnswers()["43296"].(*segmentedControlAnswer).Answer.potentialAnswerID())
	test.Equals(t, "Not sure what strength it was.", mca.Answers[0].subAnswers()["43297"].(*freeTextAnswer).Text)
	test.Equals(t, "Testing", mca.Answers[1].text())
}

func TestMultipleChoice_unmarshalProtobuf(t *testing.T) {
	pb := &intake.MultipleChoicePatientAnswer{
		AnswerSelections: []*intake.MultipleChoicePatientAnswer_Selection{
			{
				Text:              proto.String("Custom answer 1"),
				PotentialAnswerId: proto.String("9"),
			},
			{
				PotentialAnswerId: proto.String("10"),
			},
		},
	}

	data, err := proto.Marshal(pb)
	if err != nil {
		t.Fatal(err)
	}

	var mca multipleChoiceAnswer
	if err := mca.unmarshalProtobuf(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 2, len(mca.Answers))
	test.Equals(t, "Custom answer 1", mca.Answers[0].text())
	test.Equals(t, "9", mca.Answers[0].potentialAnswerID())
	test.Equals(t, "10", mca.Answers[1].potentialAnswerID())
}

func TestMultipleChoice_transformToProtoBuf(t *testing.T) {
	mca := &multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				Text:              "Option 1",
				PotentialAnswerID: "1",
			},
			&multipleChoiceAnswerSelection{
				Text:              "Option 2",
				PotentialAnswerID: "2",
			},
			&multipleChoiceAnswerSelection{
				Text:              "Option 3",
				PotentialAnswerID: "3",
			},
		},
	}

	pb, err := mca.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}

	m, ok := pb.(*intake.MultipleChoicePatientAnswer)
	if !ok {
		t.Fatalf("Expected type intake.MultipleChoicePatientAnswer but got %T", pb)
	}

	test.Equals(t, 3, len(m.AnswerSelections))
	test.Equals(t, "Option 1", *m.AnswerSelections[0].Text)
	test.Equals(t, "Option 2", *m.AnswerSelections[1].Text)
	test.Equals(t, "Option 3", *m.AnswerSelections[2].Text)
}

func TestMultipleChoice_transformForClient(t *testing.T) {
	expectedJSON := `{"type":"q_type_multiple_choice","potential_answers":[{"id":"1","text":"Option 1","answers":{"12321451251":{"type":"q_type_multiple_choice","potential_answers":[{"id":"2","text":"Option 1a"}]}}},{"id":"2","text":"Option 2"}]}`
	mca := &multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				Text:              "Option 1",
				PotentialAnswerID: "1",
				subScreens: []screen{
					&questionScreen{
						Questions: []question{
							&multipleChoiceQuestion{
								questionInfo: &questionInfo{
									ID: "12321451251",
								},
								answer: &multipleChoiceAnswer{
									Answers: []topLevelAnswerItem{
										&multipleChoiceAnswerSelection{
											Text:              "Option 1a",
											PotentialAnswerID: "2",
										},
									},
								},
							},
						},
					},
				},
			},
			&multipleChoiceAnswerSelection{
				Text:              "Option 2",
				PotentialAnswerID: "2",
			},
		},
	}

	data, err := mca.transformForClient()
	if err != nil {
		t.Fatal(err)
	}

	jsonData, err := json.Marshal(data)
	test.OK(t, err)

	if expectedJSON != string(jsonData) {
		t.Fatalf("Expected `%s` but got `%s`", expectedJSON, string(jsonData))
	}
}

func TestMultipleChoice_equals(t *testing.T) {
	mca := &multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				Text:              "Option 1",
				PotentialAnswerID: "1",
				SubAnswers: map[string]patientAnswer{
					"12321451251": &multipleChoiceAnswer{
						Answers: []topLevelAnswerItem{
							&multipleChoiceAnswerSelection{
								Text:              "Option 1a",
								PotentialAnswerID: "2",
							},
						},
					},
				},
			},
			&multipleChoiceAnswerSelection{
				Text:              "Option 2",
				PotentialAnswerID: "2",
			},
		},
	}

	if !mca.equals(mca) {
		t.Fatal("Expected same answer to match itself")
	}

	// answer should match even if subanswers are different cause subanswers
	// should not play a role in the equality
	other := &multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				Text:              "Option 1",
				PotentialAnswerID: "1",
			},
			&multipleChoiceAnswerSelection{
				Text:              "Option 2",
				PotentialAnswerID: "2",
			},
		},
	}

	if !mca.equals(other) {
		t.Fatal("Answer expected to match even if subanswers don't match")
	}

	// answer should match without subanswers
	mca = &multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				Text:              "Option 1",
				PotentialAnswerID: "1",
			},
			&multipleChoiceAnswerSelection{
				Text:              "Option 2",
				PotentialAnswerID: "2",
			},
		},
	}

	if !mca.equals(other) {
		t.Fatal("Expected same answer to match itself")
	}

	// answer should not match when order differs
	other = &multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				Text:              "Option 2",
				PotentialAnswerID: "2",
			},
			&multipleChoiceAnswerSelection{
				Text:              "Option 1",
				PotentialAnswerID: "1",
			},
		},
	}

	if mca.equals(other) {
		t.Fatal("Answer not expected to match")
	}

	// answer should not match when options differ
	other = &multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				Text:              "Option 2",
				PotentialAnswerID: "2",
			},
		},
	}

	if mca.equals(other) {
		t.Fatal("Answer not expected to match")
	}

}
