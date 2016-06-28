package manager

type answerCondition struct {
	Op                 string   `json:"op"`
	QuestionID         string   `json:"question_id"`
	PotentialAnswersID []string `json:"potential_answers_id"`

	dependencies []layoutUnit
}

func (a *answerCondition) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys(
		"answer_conditon",
		"op", "question_id", "potential_answers_id"); err != nil {
		return err
	}

	a.Op = data.mustGetString("op")
	a.QuestionID = data.mustGetString("question_id")

	var err error
	a.PotentialAnswersID, err = data.getStringSlice("potential_answers_id")

	return err
}

func (a *answerCondition) questionIDs() []string {
	return []string{a.QuestionID}
}

func (a *answerCondition) layoutUnitDependencies(dataSource questionAnswerDataSource) []layoutUnit {
	if a.dependencies == nil {
		if q := dataSource.question(a.QuestionID); q != nil {
			a.dependencies = []layoutUnit{q}
		}
	}

	return a.dependencies
}

func (a *answerCondition) staticInfoCopy(context map[string]string) interface{} {
	conditionCopy := &answerCondition{
		Op:                 a.Op,
		QuestionID:         a.QuestionID,
		PotentialAnswersID: make([]string, len(a.PotentialAnswersID)),
	}

	copy(conditionCopy.PotentialAnswersID, a.PotentialAnswersID)

	return conditionCopy
}
