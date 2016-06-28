package manager

// logicalCondition represents a general
// group of conditions that share inputs (and, or, not)
type logicalCondition struct {
	Op       string      `json:"op"`
	Operands []condition `json:"operands:"`

	dependencies []layoutUnit
}

func (a *logicalCondition) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("logical_condition", "op", "operands"); err != nil {
		return err
	}

	a.Op = data.mustGetString("op")
	operands, err := data.getInterfaceSlice("operands")
	if err != nil {
		return err
	}

	a.Operands = make([]condition, len(operands))
	for i, operand := range operands {

		operandData, err := getDataMap(operand)
		if err != nil {
			return err
		}

		a.Operands[i], err = getCondition(operandData)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *logicalCondition) questionIDs() []string {
	var questionIDs []string
	for _, operand := range a.Operands {
		qIDs := operand.questionIDs()
		if len(qIDs) > 0 {
			questionIDs = append(questionIDs, qIDs...)
		}
	}

	return questionIDs
}

func (a *logicalCondition) layoutUnitDependencies(dataSource questionAnswerDataSource) []layoutUnit {
	if a.dependencies == nil {
		for _, operand := range a.Operands {
			if len(operand.layoutUnitDependencies(dataSource)) > 0 {
				a.dependencies = append(a.dependencies, operand.layoutUnitDependencies(dataSource)...)
			}
		}
	}

	return a.dependencies
}

func (a *logicalCondition) staticInfoCopy(context map[string]string) interface{} {
	conditionCopy := &logicalCondition{
		Op:       a.Op,
		Operands: make([]condition, len(a.Operands)),
	}

	for i, operand := range a.Operands {
		conditionCopy.Operands[i] = operand.staticInfoCopy(context).(condition)
	}

	return conditionCopy
}
