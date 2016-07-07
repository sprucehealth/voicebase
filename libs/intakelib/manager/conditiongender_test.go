package manager

import "testing"

type mockDataSource_genderCondition struct {
	gender string
	questionAnswerDataSource
}

func (m *mockDataSource_genderCondition) question(questionID string) question {
	return nil
}

func (m *mockDataSource_genderCondition) valueForKey(key string) interface{} {
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
	m.gender = "male"
	if c.evaluate(m) {
		t.Fatalf("Expected condition to evaluate to false but evaluated to true")
	}

	m.gender = "other"
	if c.evaluate(m) {
		t.Fatalf("Expected condition to evaluate to false but evaluated to true")
	}

	// should evaluate to true when the value for the key is indeed the expected value
	m.gender = "female"
	if !c.evaluate(m) {
		t.Fatalf("Expected condition to evaluate to true but it didnt")
	}

	// evaluation should be case insensitive
	m.gender = "Female"
	if !c.evaluate(m) {
		t.Fatalf("Expected condition to evaluate to true but it didnt")
	}
}
