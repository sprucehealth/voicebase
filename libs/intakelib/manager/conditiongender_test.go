package manager

import "testing"

type mockDataSource_genderCondition struct {
	gender []byte
	questionAnswerDataSource
}

func (m *mockDataSource_genderCondition) question(questionID string) question {
	return nil
}

func (m *mockDataSource_genderCondition) valueForKey(key string) []byte {
	return m.gender
}

func TestConditionGender_NoKey(t *testing.T) {
	m := &mockDataSource_genderCondition{}

	c := &genderCondition{
		Op:     "gender_equals",
		Gender: "female",
	}

	// should evaluate to false when the key is not present
	if c.evaluate(m) {
		t.Fatalf("Expected condition to evaluate to false but evaluated to true")
	}

	// should evaluate to false when the key present is not the expected value
	m.gender = []byte("male")
	if c.evaluate(m) {
		t.Fatalf("Expected condition to evaluate to false but evaluated to true")
	}

	m.gender = []byte("other")
	if c.evaluate(m) {
		t.Fatalf("Expected condition to evaluate to false but evaluated to true")
	}

	// should evaluate to true when the value for the key is indeed the expected value
	m.gender = []byte("female")
	if !c.evaluate(m) {
		t.Fatalf("Expected condition to evaluate to true but it didnt")
	}

	// evaluation should be case insensitive
	m.gender = []byte("Female")
	if !c.evaluate(m) {
		t.Fatalf("Expected condition to evaluate to true but it didnt")
	}
}
