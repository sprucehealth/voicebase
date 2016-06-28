package manager

import "fmt"

type notCondition struct {
	logicalCondition
}

func (a *notCondition) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys(conditionTypeAND.String(), "op", "operands"); err != nil {
		return err
	}

	a.Op = data.mustGetString("op")
	operands, err := data.getInterfaceSlice("operands")
	if err != nil {
		return err
	}

	if len(operands) != 1 {
		return fmt.Errorf("Expected single operand for NOT condition but got %d", len(operands))
	}

	a.Operands = make([]condition, 1)
	operandData, err := getDataMap(operands[0])
	if err != nil {
		return err
	}

	a.Operands[0], err = getCondition(operandData)

	return err
}

func (a *notCondition) evaluate(dataSource questionAnswerDataSource) bool {
	return !a.Operands[0].evaluate(dataSource)
}

func (a *notCondition) staticInfoCopy(context map[string]string) interface{} {
	return &notCondition{
		logicalCondition: *(a.logicalCondition.staticInfoCopy(context).(*logicalCondition)),
	}
}
