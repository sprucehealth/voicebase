package manager

type answerContainsAnyCondition struct {
	answerCondition
}

func (a *answerContainsAnyCondition) evaluate(dataSource questionAnswerDataSource) bool {
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

	// patient answer must contain any of the potential answers specified in the condition
	for _, pID := range a.PotentialAnswersID {
		for _, answerItem := range answerContainer.topLevelAnswers() {
			if answerItem.potentialAnswerID() == pID {
				return true
			}
		}
	}

	return false
}

func (a *answerContainsAnyCondition) staticInfoCopy(context map[string]string) interface{} {
	return &answerContainsAnyCondition{
		answerCondition: *(a.answerCondition.staticInfoCopy(context).(*answerCondition)),
	}
}
