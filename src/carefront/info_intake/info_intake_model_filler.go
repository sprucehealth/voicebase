package info_intake

import (
	"carefront/api"
	"strconv"
)

func (c *Condition) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	if c.QuestionTag == "" {
		return nil
	}
	questionId, _, _, err := dataApi.GetQuestionInfo(c.QuestionTag, languageId)
	if err != nil {
		return err
	}
	c.QuestionId = questionId
	c.PotentialAnswersId = make([]string, len(c.PotentialAnswersTags))
	for i, tag := range c.PotentialAnswersTags {
		answerIds, _, _, answerTags, _, err := dataApi.GetAnswerInfo(questionId, languageId)
		if err != nil {
			return err
		}
		for j, answerTag := range answerTags {
			if answerTag == tag {
				c.PotentialAnswersId[i] = strconv.Itoa(int(answerIds[j]))
				break
			}
		}
	}
	return nil
}

func (t *TipSection) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	_, tipSectionTitle, tipSectionSubtext, err := dataApi.GetTipSectionInfo(t.TipsSectionTag, languageId)
	if err != nil {
		return err
	}

	t.TipsSectionTitle = tipSectionTitle
	t.TipsSubtext = tipSectionSubtext

	t.Tips = make([]string, len(t.TipsTags))
	for i, tipTag := range t.TipsTags {
		_, tipText, err := dataApi.GetTipInfo(tipTag, languageId)
		if err != nil {
			return err
		}
		t.Tips[i] = tipText
	}

	return nil
}

func (q *Question) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	questionId, questionTitle, questionType, err := dataApi.GetQuestionInfo(q.QuestionTag, languageId)
	if err != nil {
		return err
	}
	q.QuestionId = questionId
	q.QuestionTitle = questionTitle
	q.QuestionType = questionType

	// go over the potential ansnwer tags to create potential outcome blocks
	q.PotentialAnswers = make([]*PotentialAnswer, 0, 5)
	answerIds, answers, answerTypes, _, orderings, err := dataApi.GetAnswerInfo(questionId, languageId)
	if err != nil {
		return err
	}

	for i, _ := range answerIds {
		potentialAnswer := &PotentialAnswer{AnswerId: answerIds[i],
			Answer:     answers[i],
			AnswerType: answerTypes[i],
			Ordering:   orderings[i]}
		q.PotentialAnswers = append(q.PotentialAnswers, potentialAnswer)
	}

	if q.ConditionBlock != nil {
		err := q.ConditionBlock.FillInDatabaseInfo(dataApi, languageId)
		if err != nil {
			return err
		}
	}

	if q.Tips != nil {
		err := q.Tips.FillInDatabaseInfo(dataApi, languageId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Screen) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	if s.ConditionBlock != nil {
		err := s.ConditionBlock.FillInDatabaseInfo(dataApi, languageId)
		if err != nil {
			return err
		}
	}

	if s.Questions != nil {
		for _, question := range s.Questions {
			err := question.FillInDatabaseInfo(dataApi, languageId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Section) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	sectionId, sectionTitle, err := dataApi.GetSectionInfo(s.SectionTag, languageId)
	if err != nil {
		return err
	}
	s.SectionId = sectionId
	s.SectionTitle = sectionTitle
	for _, screen := range s.Screens {
		err := screen.FillInDatabaseInfo(dataApi, languageId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *HealthCondition) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	healthConditionId, err := dataApi.GetHealthConditionInfo(t.HealthConditionTag)
	if err != nil {
		return err
	}
	t.HealthConditionId = healthConditionId
	for _, section := range t.Sections {
		err := section.FillInDatabaseInfo(dataApi, languageId)
		if err != nil {
			return err
		}
	}
	return nil
}
