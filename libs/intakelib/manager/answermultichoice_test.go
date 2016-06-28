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
		"answers": [
			{
				"answer_id": "64439",
				"question_id": "43301",
				"potential_answer_id": "143498",
				"potential_answer": "Benzoyl peroxide",
				"potential_answer_summary": "Benzoyl peroxide",
				"type" : "q_type_multiple_choice",
				"answers": [{
					"answer_id": "64440",
					"question_id": "43295",
					"potential_answer_id": "143487",
					"potential_answer": "No",
					"potential_answer_summary": "No",
					"type" : "q_type_single_select"
				}, {
					"answer_id": "64441",
					"question_id": "43296",
					"potential_answer_id": "143489",
					"potential_answer": "Somewhat",
					"potential_answer_summary": "Somewhat",
					"type": "q_type_segmented_control"
				}, {
					"answer_id": "64442",
					"question_id": "43297",
					"potential_answer_id": null,
					"answer_text": "Not sure what strength it was.",
					"type" : "q_type_free_text"
				}]
			},
			{
				"answer_id": "64439",
				"question_id": "43301",
				"answer_text" : "Testing",
				"type" : "q_type_multiple_choice"
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
	test.Equals(t, "143487", mca.Answers[0].subAnswers()[0].(*multipleChoiceAnswer).Answers[0].potentialAnswerID())
	test.Equals(t, "143489", mca.Answers[0].subAnswers()[1].(*multipleChoiceAnswer).Answers[0].potentialAnswerID())
	test.Equals(t, "Not sure what strength it was.", mca.Answers[0].subAnswers()[2].(*freeTextAnswer).Text)
	test.Equals(t, "Testing", mca.Answers[1].text())

	// test alternate parsing of single object for multiple choice

	clientJSON = `
	{
		"answer_id": "64439",
		"question_id": "43301",
		"answer_text" : "Testing",
		"type" : "q_type_multiple_choice"
	}`

	data = make(map[string]interface{})
	if err := json.Unmarshal([]byte(clientJSON), &data); err != nil {
		t.Fatal(err)
	}

	mca = multipleChoiceAnswer{}
	if err := mca.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 1, len(mca.Answers))
	test.Equals(t, "Testing", mca.Answers[0].text())
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
		QuestionID: "1234355",
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

func TestMultipleChoice_marshalJSONForClient(t *testing.T) {
	expectedJSON := `{"question_id":"1234355","potential_answers":[{"potential_answer_id":"1","answer_text":"Option 1","answers":[{"question_id":"12321451251","potential_answers":[{"potential_answer_id":"2","answer_text":"Option 1a"}]}]},{"potential_answer_id":"2","answer_text":"Option 2"}]}`
	mca := &multipleChoiceAnswer{
		QuestionID: "1234355",
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				Text:              "Option 1",
				PotentialAnswerID: "1",
				subScreens: []screen{
					&questionScreen{
						Questions: []question{
							&multipleChoiceQuestion{
								questionInfo: &questionInfo{},
								answer: &multipleChoiceAnswer{
									QuestionID: "12321451251",
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

	jsonData, err := mca.marshalJSONForClient()
	if err != nil {
		t.Fatal(err)
	}

	if expectedJSON != string(jsonData) {
		t.Fatalf("Expected `%s` but got `%s`", expectedJSON, string(jsonData))
	}
}

func TestMultipleChoice_equals(t *testing.T) {
	mca := &multipleChoiceAnswer{
		QuestionID: "1234355",
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				Text:              "Option 1",
				PotentialAnswerID: "1",
				SubAnswers: []patientAnswer{
					&multipleChoiceAnswer{
						QuestionID: "12321451251",
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
		QuestionID: "1234355",
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
		QuestionID: "1234355",
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
		QuestionID: "1234355",
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
		QuestionID: "1234355",
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
