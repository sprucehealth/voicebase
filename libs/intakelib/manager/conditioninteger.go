package manager

import (
	"github.com/sprucehealth/backend/libs/errors"
)

type integerCondition struct {
	Op         string `json:"op"`
	IntValue   int    `json:"int_value"`
	DataSource string `json:"data_source"`
}

func (i *integerCondition) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("integer_condition", "op", "int_value", "data_source"); err != nil {
		return errors.Trace(err)
	}

	i.Op = data.mustGetString("op")
	i.IntValue = data.mustGetInt("int_value")
	i.DataSource = data.mustGetString("data_source")

	return nil
}

func (i *integerCondition) questionIDs() []string {
	return nil
}

func (i *integerCondition) evaluate(dataSource questionAnswerDataSource) bool {
	data := dataSource.valueForKey(i.DataSource)
	intValue, ok := data.(int)
	if !ok {
		return false
	}

	switch i.Op {
	case conditionTypeIntegerEqualTo.String():
		return i.IntValue == intValue
	case conditionTypeIntegerGreaterThan.String():
		return intValue > i.IntValue
	case conditionTypeIntegerGreaterThanEqualTo.String():
		return intValue >= i.IntValue
	case conditionTypeIntegerLessThan.String():
		return intValue < i.IntValue
	case conditionTypeIntegerLessThanEqualTo.String():
		return intValue <= i.IntValue

	}

	return false

}

func (a *integerCondition) layoutUnitDependencies(dataSource questionAnswerDataSource) []layoutUnit {
	return nil
}

func (i *integerCondition) staticInfoCopy(context map[string]string) interface{} {
	return &integerCondition{
		Op:         i.Op,
		DataSource: i.DataSource,
		IntValue:   i.IntValue,
	}
}
