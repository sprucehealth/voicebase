package manager

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

type mockDataSource_integerCondition struct {
	intValue int
	questionAnswerDataSource
}

func (m *mockDataSource_integerCondition) question(questionID string) question {
	return nil
}

func (m *mockDataSource_integerCondition) valueForKey(key string) interface{} {
	if key == keyTypePatientAgeInYears.String() {
		return m.intValue
	}
	return 0
}

func TestIntegerCondition_Equal(t *testing.T) {
	m := &mockDataSource_integerCondition{}

	c := &integerCondition{
		Op:         conditionTypeIntegerEqualTo.String(),
		IntValue:   10,
		DataSource: "age_in_years",
	}

	m.intValue = 11
	test.Equals(t, false, c.evaluate(m))

	m.intValue = 10
	test.Equals(t, true, c.evaluate(m))
}

func TestIntegerCondition_LessThan(t *testing.T) {
	m := &mockDataSource_integerCondition{}

	c := &integerCondition{
		Op:         conditionTypeIntegerLessThan.String(),
		IntValue:   10,
		DataSource: "age_in_years",
	}

	m.intValue = 11
	test.Equals(t, false, c.evaluate(m))

	m.intValue = 9
	test.Equals(t, true, c.evaluate(m))

	m.intValue = 10
	test.Equals(t, false, c.evaluate(m))
}

func TestIntegerCondition_LessThanEqualTo(t *testing.T) {
	m := &mockDataSource_integerCondition{}

	c := &integerCondition{
		Op:         conditionTypeIntegerLessThanEqualTo.String(),
		IntValue:   10,
		DataSource: "age_in_years",
	}

	m.intValue = 11
	test.Equals(t, false, c.evaluate(m))

	m.intValue = 9
	test.Equals(t, true, c.evaluate(m))

	m.intValue = 10
	test.Equals(t, true, c.evaluate(m))
}

func TestIntegerCondition_GreaterThan(t *testing.T) {
	m := &mockDataSource_integerCondition{}

	c := &integerCondition{
		Op:         conditionTypeIntegerGreaterThan.String(),
		IntValue:   10,
		DataSource: "age_in_years",
	}

	m.intValue = 9
	test.Equals(t, false, c.evaluate(m))

	m.intValue = 11
	test.Equals(t, true, c.evaluate(m))

	m.intValue = 10
	test.Equals(t, false, c.evaluate(m))
}

func TestIntegerCondition_GreaterThanEqualTo(t *testing.T) {
	m := &mockDataSource_integerCondition{}

	c := &integerCondition{
		Op:         conditionTypeIntegerGreaterThanEqualTo.String(),
		IntValue:   10,
		DataSource: "age_in_years",
	}

	m.intValue = 9
	test.Equals(t, false, c.evaluate(m))

	m.intValue = 11
	test.Equals(t, true, c.evaluate(m))

	m.intValue = 10
	test.Equals(t, true, c.evaluate(m))
}
