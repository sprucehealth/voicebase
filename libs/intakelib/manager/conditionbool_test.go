package manager

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

type mockDataSource_boolCondition struct {
	boolValue bool
	questionAnswerDataSource
}

func (m *mockDataSource_boolCondition) question(questionID string) question {
	return nil
}

func (m *mockDataSource_boolCondition) valueForKey(key string) interface{} {
	return m.boolValue
}

func TestBoolCondition(t *testing.T) {
	m := &mockDataSource_boolCondition{}

	c := &boolCondition{
		Op:         conditionTypeBooleanEquals.String(),
		BoolValue:  false,
		DataSource: "preference.optional_triage",
	}

	m.boolValue = true
	test.Equals(t, false, c.evaluate(m))

	m.boolValue = false
	test.Equals(t, true, c.evaluate(m))
}
