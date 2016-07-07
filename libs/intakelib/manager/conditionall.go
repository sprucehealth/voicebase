package manager

type answerContainsAllCondition struct {
	answerCondition
}

func (a *answerContainsAllCondition) evaluate(dataSource questionAnswerDataSource) bool {
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

	// patient answer must contain all the potential answers specified in the condition
	for _, pID := range a.PotentialAnswersID {

		var found bool
		for _, a := range answerContainer.topLevelAnswers() {
			if a.potentialAnswerID() == pID {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func (a *answerContainsAllCondition) staticInfoCopy(context map[string]string) interface{} {
	return &answerContainsAllCondition{
		answerCondition: *(a.answerCondition.staticInfoCopy(context).(*answerCondition)),
	}
}
