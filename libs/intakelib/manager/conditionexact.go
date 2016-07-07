package manager

type answerEqualsExactCondition struct {
	answerCondition
}

func (a *answerEqualsExactCondition) evaluate(dataSource questionAnswerDataSource) bool {
	q := dataSource.question(a.QuestionID)
	if q.visibility() == hidden {
		return false
	}
	mcq, ok := q.(*multipleChoiceQuestion)
	if !ok {
		return false
	}

	pa, err := mcq.patientAnswer()
	if err != nil {
		return false
	}

	answerContainer, ok := pa.(topLevelAnswerWithSubScreensContainer)
	if !ok {
		return false
	}

	// patient answer must contain exactly the same answers specified in the condition
	if len(answerContainer.topLevelAnswers()) != len(a.PotentialAnswersID) {
		return false
	}

	answerIDsFoundOnce := make(map[string]bool, len(a.PotentialAnswersID))
	for _, aItem := range answerContainer.topLevelAnswers() {
		answerIDsFoundOnce[aItem.potentialAnswerID()] = !answerIDsFoundOnce[aItem.potentialAnswerID()]
	}

	for _, pID := range a.PotentialAnswersID {
		if !answerIDsFoundOnce[pID] {
			return false
		}
	}

	return true
}

func (a *answerEqualsExactCondition) staticInfoCopy(context map[string]string) interface{} {
	return &answerEqualsExactCondition{
		answerCondition: *(a.answerCondition.staticInfoCopy(context).(*answerCondition)),
	}
}
