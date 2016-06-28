package manager

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

const conditionAnyJSON = `
{
	"op": "answer_contains_any",
	"type": "answer_contains_any",
	"question": "q_regularly_taking_medications",
	"question_id": "40637",
	"potential_answers_id": ["126596"],
	"potential_answers": ["q_regularly_taking_medications_yes"]
}`

func TestAnswerConditionParsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(conditionAnyJSON), &data); err != nil {
		t.Fatal(err)
	}

	ca := &answerContainsAllCondition{}
	if err := ca.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "answer_contains_any", ca.Op)
	test.Equals(t, "40637", ca.QuestionID)
	test.Equals(t, []string{"126596"}, ca.PotentialAnswersID)
}

type mockDataSource_answerCondition struct {
	q question
	questionAnswerDataSource
}

func (m *mockDataSource_answerCondition) question(questionID string) question {
	return m.q
}

func TestAnswerCondition_staticInfoCopy(t *testing.T) {
	// answer contains all condition
	ca := &answerContainsAllCondition{
		answerCondition: answerCondition{
			QuestionID:         "123",
			Op:                 "answer_contains_any",
			PotentialAnswersID: []string{"12"},
		},
	}

	ca1 := ca.staticInfoCopy(nil).(*answerContainsAllCondition)
	test.Equals(t, ca, ca1)

	// answer contains any condition
	cany := &answerContainsAnyCondition{
		answerCondition: answerCondition{
			QuestionID:         "1",
			PotentialAnswersID: []string{"10", "100", "101"},
		},
	}

	cany2 := cany.staticInfoCopy(nil).(*answerContainsAnyCondition)
	test.Equals(t, cany, cany2)

	// answer contains exact condition
	cexact := &answerEqualsExactCondition{
		answerCondition: answerCondition{
			QuestionID:         "1",
			PotentialAnswersID: []string{"10", "11", "12"},
		},
	}

	cexact2 := cexact.staticInfoCopy(nil).(*answerEqualsExactCondition)
	test.Equals(t, cexact, cexact2)
}

func TestConditionAll_evaluate(t *testing.T) {
	m := &mockDataSource_answerCondition{
		q: &multipleChoiceQuestion{
			questionInfo: &questionInfo{},
			answer: &multipleChoiceAnswer{
				Answers: []topLevelAnswerItem{
					&multipleChoiceAnswerSelection{
						PotentialAnswerID: "10",
					},
					&multipleChoiceAnswerSelection{
						PotentialAnswerID: "11",
					},
					&multipleChoiceAnswerSelection{
						PotentialAnswerID: "12",
					},
				},
			},
		},
	}

	a := &answerContainsAllCondition{
		answerCondition: answerCondition{
			QuestionID:         "1",
			PotentialAnswersID: []string{"10", "11", "12"},
		},
	}

	if !a.evaluate(m) {
		t.Fatal("Expected evaluation to succeed but it didn't")
	}

	// change question to be hidden and condition should evaluate to false
	m.q.setVisibility(hidden)

	if a.evaluate(m) {
		t.Fatal("Expected evaluation to fail but it didn't")
	}

	m.q.setVisibility(visible)

	a = &answerContainsAllCondition{
		answerCondition: answerCondition{
			QuestionID:         "1",
			PotentialAnswersID: []string{"10", "11", "12", "13"},
		},
	}

	if a.evaluate(m) {
		t.Fatal("Expected evaluation to fail but it didn't")
	}
}

func TestConditionAny_evaluate(t *testing.T) {
	m := &mockDataSource_answerCondition{
		q: &multipleChoiceQuestion{
			questionInfo: &questionInfo{},
			answer: &multipleChoiceAnswer{
				Answers: []topLevelAnswerItem{
					&multipleChoiceAnswerSelection{
						PotentialAnswerID: "10",
					},
					&multipleChoiceAnswerSelection{
						PotentialAnswerID: "11",
					},
					&multipleChoiceAnswerSelection{
						PotentialAnswerID: "12",
					},
				},
			},
		},
	}

	a := &answerContainsAnyCondition{
		answerCondition: answerCondition{
			QuestionID:         "1",
			PotentialAnswersID: []string{"10", "100", "101"},
		},
	}

	if !a.evaluate(m) {
		t.Fatal("Expected evaluation to succeed but it didn't")
	}

	// change question to be hidden and condition should evaluate to false
	m.q.setVisibility(hidden)

	if a.evaluate(m) {
		t.Fatal("Expected evaluation to fail but it didn't")
	}

	m.q.setVisibility(visible)

	a = &answerContainsAnyCondition{
		answerCondition: answerCondition{
			QuestionID:         "1",
			PotentialAnswersID: []string{"20"},
		},
	}

	if a.evaluate(m) {
		t.Fatal("Expected evaluation to fail but it didn't")
	}
}

func TestConditionExact_evaluate(t *testing.T) {
	m := &mockDataSource_answerCondition{
		q: &multipleChoiceQuestion{
			questionInfo: &questionInfo{},
			answer: &multipleChoiceAnswer{
				Answers: []topLevelAnswerItem{
					&multipleChoiceAnswerSelection{
						PotentialAnswerID: "10",
					},
					&multipleChoiceAnswerSelection{
						PotentialAnswerID: "11",
					},
					&multipleChoiceAnswerSelection{
						PotentialAnswerID: "12",
					},
				},
			},
		},
	}

	a := &answerEqualsExactCondition{
		answerCondition: answerCondition{
			QuestionID:         "1",
			PotentialAnswersID: []string{"10", "11", "12"},
		},
	}

	if !a.evaluate(m) {
		t.Fatal("Expected evaluation to succeed but it didn't")
	}

	// change question to be hidden and condition should evaluate to false
	m.q.setVisibility(hidden)

	if a.evaluate(m) {
		t.Fatal("Expected evaluation to fail but it didn't")
	}

	m.q.setVisibility(visible)

	a = &answerEqualsExactCondition{
		answerCondition: answerCondition{
			QuestionID:         "1",
			PotentialAnswersID: []string{"11", "12", "10"},
		},
	}

	if !a.evaluate(m) {
		t.Fatal("Expected evaluation to succeed but it didn't")
	}

	a = &answerEqualsExactCondition{
		answerCondition: answerCondition{
			QuestionID:         "1",
			PotentialAnswersID: []string{"10", "11", "12", "13"},
		},
	}

	if a.evaluate(m) {
		t.Fatal("Expected evaluation to fail but it didn't")
	}
}
