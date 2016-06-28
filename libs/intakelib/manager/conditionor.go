package manager

type orCondition struct {
	logicalCondition
}

func (a *orCondition) evaluate(dataSource questionAnswerDataSource) bool {
	// all conditions have to evaluate to true
	for _, operand := range a.Operands {
		if operand.evaluate(dataSource) {
			return true
		}
	}

	return false
}

func (a *orCondition) staticInfoCopy(context map[string]string) interface{} {
	return &orCondition{
		logicalCondition: *(a.logicalCondition.staticInfoCopy(context).(*logicalCondition)),
	}
}
