package manager

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

const conditionANDJson = `
{
	"op": "and",
	"type": "and",
	"operands": [{
		"op": "answer_contains_any",
		"type": "answer_contains_any",
		"question": "q_derm_rash_affected_areas",
		"question_id": "40551",
		"potential_answers_id": ["126308"],
		"potential_answers": ["a_derm_rash_affected_areas_mouth"]
	}, {
		"op": "answer_contains_any",
		"type": "answer_contains_any",
		"question": "q_derm_rash_affected_areas",
		"question_id": "40551",
		"potential_answers_id": ["126314"],
		"potential_answers": ["a_derm_rash_affected_areas_groin"]
	}]
}`

type logicTestCondition struct {
	condition
	res bool
}

func (a *logicTestCondition) evaluate(dataSource questionAnswerDataSource) bool {
	return a.res
}

func TestLogicalConditionParsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(conditionANDJson), &data); err != nil {
		t.Fatal(err)
	}

	ca := &andCondition{}
	if err := ca.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "and", ca.Op)
	test.Equals(t, 2, len(ca.Operands))
	test.Equals(t, "answer_contains_any", ca.Operands[0].(*answerContainsAnyCondition).Op)
	test.Equals(t, "answer_contains_any", ca.Operands[1].(*answerContainsAnyCondition).Op)
}

func TestLogicCondition_staticInfoCopy(t *testing.T) {
	cand := &andCondition{
		logicalCondition: logicalCondition{
			Op: "and",
			Operands: []condition{
				&answerContainsAllCondition{
					answerCondition: answerCondition{
						QuestionID:         "123",
						Op:                 "answer_contains_any",
						PotentialAnswersID: []string{"12"},
					},
				},
				&answerContainsAllCondition{
					answerCondition: answerCondition{
						QuestionID:         "12345",
						Op:                 "answer_contains_any",
						PotentialAnswersID: []string{"12"},
					},
				},
			},
		},
	}

	cand2 := cand.staticInfoCopy(nil).(*andCondition)
	test.Equals(t, cand, cand2)

	cor := &orCondition{
		logicalCondition: logicalCondition{
			Operands: []condition{
				&answerContainsAllCondition{
					answerCondition: answerCondition{
						QuestionID:         "123",
						Op:                 "answer_contains_any",
						PotentialAnswersID: []string{"12"},
					},
				},
				&answerContainsAllCondition{
					answerCondition: answerCondition{
						QuestionID:         "12345",
						Op:                 "answer_contains_any",
						PotentialAnswersID: []string{"12"},
					},
				},
			},
		},
	}

	cor2 := cor.staticInfoCopy(nil).(*orCondition)
	test.Equals(t, cor, cor2)

	cnot := &notCondition{
		logicalCondition: logicalCondition{
			Operands: []condition{
				&answerContainsAllCondition{
					answerCondition: answerCondition{
						QuestionID:         "123",
						Op:                 "answer_contains_any",
						PotentialAnswersID: []string{"12"},
					},
				},
			},
		},
	}

	cnot2 := cnot.staticInfoCopy(nil).(*notCondition)
	test.Equals(t, cnot, cnot2)
}

type mockDataSource_logicCondition struct {
	questionMap map[string]question
	questionAnswerDataSource
}

func (m *mockDataSource_logicCondition) question(questionID string) question {
	return m.questionMap[questionID]
}

func TestLogicCondition_dependancies(t *testing.T) {
	ca := &andCondition{
		logicalCondition: logicalCondition{
			Op: "and",
			Operands: []condition{
				&answerContainsAllCondition{
					answerCondition: answerCondition{
						QuestionID:         "123",
						Op:                 "answer_contains_any",
						PotentialAnswersID: []string{"12"},
					},
				},
				&answerContainsAllCondition{
					answerCondition: answerCondition{
						QuestionID:         "12345",
						Op:                 "answer_contains_any",
						PotentialAnswersID: []string{"12"},
					},
				},
			},
		},
	}

	m := &mockDataSource_logicCondition{
		questionMap: map[string]question{
			"123":   &freeTextQuestion{},
			"12345": &autocompleteQuestion{},
		},
	}

	test.Equals(t, 2, len(ca.layoutUnitDependencies(m)))
	test.Equals(t, m.questionMap["123"], ca.layoutUnitDependencies(m)[0])
	test.Equals(t, m.questionMap["12345"], ca.layoutUnitDependencies(m)[1])
}

func TestConditionAnd_evaluate(t *testing.T) {

	a1 := &logicTestCondition{
		res: true,
	}

	a2 := &logicTestCondition{
		res: true,
	}

	aCondition := &andCondition{

		logicalCondition: logicalCondition{
			Operands: []condition{a1, a2},
		},
	}

	if !aCondition.evaluate(nil) {
		t.Fatal("Expected evaluation to succeed but it didn't")
	}

	a3 := &logicTestCondition{
		res: false,
	}

	aCondition.Operands = append(aCondition.Operands, a3)

	if aCondition.evaluate(nil) {
		t.Fatal("Expected evaluation to fail but it didn't")
	}
}

func TestConditionOR_evaluate(t *testing.T) {

	a1 := &logicTestCondition{
		res: true,
	}

	a2 := &logicTestCondition{
		res: false,
	}

	aCondition := &orCondition{

		logicalCondition: logicalCondition{
			Operands: []condition{a1, a2},
		},
	}

	if !aCondition.evaluate(nil) {
		t.Fatal("Expected evaluation to succeed but it didn't")
	}

	a1.res = false
	if aCondition.evaluate(nil) {
		t.Fatal("Expected evaluation to fail but it didn't")
	}

}

func TestConditionNOT_evaluate(t *testing.T) {
	a1 := &logicTestCondition{
		res: false,
	}

	aCondition := &notCondition{

		logicalCondition: logicalCondition{
			Operands: []condition{a1},
		},
	}

	if !aCondition.evaluate(nil) {
		t.Fatal("Expected evaluation to succeed but it didn't")
	}

	a1.res = true
	if aCondition.evaluate(nil) {
		t.Fatal("Expected evaluation to fail but it didn't")
	}

}
