package manager

type andCondition struct {
	logicalCondition
}

func (a *andCondition) evaluate(dataSource questionAnswerDataSource) bool {

	// all conditions have to evaluate to true
	for _, operand := range a.Operands {
		if !operand.evaluate(dataSource) {
			return false
		}
	}

	return true
}

func (a *andCondition) staticInfoCopy(context map[string]string) interface{} {
	return &andCondition{
		logicalCondition: *(a.logicalCondition.staticInfoCopy(context).(*logicalCondition)),
	}
}
