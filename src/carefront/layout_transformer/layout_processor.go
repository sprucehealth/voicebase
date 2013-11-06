package layout_transformer

import (
	"carefront/api"
	"strconv"
)

const (
	EN_LANGUAGE_ID = 1
)

func (c *Condition) FillInDatabaseInfo(dataApi *api.DataService) error {
	if c.QuestionTag == "" {
		return nil
	}
	questionId, _, _, err := dataApi.GetQuestionInfo(c.QuestionTag, EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	c.QuestionId = questionId
	c.PotentialAnswersId = make([]string, len(c.PotentialAnswersTags))
	for i, tag := range c.PotentialAnswersTags {
		answerId, _, _, err := dataApi.GetOutcomeInfo(tag, EN_LANGUAGE_ID)
		c.PotentialAnswersId[i] = strconv.Itoa(int(answerId))
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *TipSection) FillInDatabaseInfo(dataApi *api.DataService) error {
	_, tipSectionTitle, tipSectionSubtext, err := dataApi.GetTipSectionInfo(t.TipsSectionTag, EN_LANGUAGE_ID)
	if err != nil {
		return err
	}

	t.TipsSectionTitle = tipSectionTitle
	t.TipsSubtext = tipSectionSubtext

	t.Tips = make([]string, len(t.TipsTags))
	for i, tipTag := range t.TipsTags {
		_, tipText, err := dataApi.GetTipInfo(tipTag, EN_LANGUAGE_ID)
		if err != nil {
			return err
		}
		t.Tips[i] = tipText
	}

	return nil
}

func (q *Question) FillInDatabaseInfo(dataApi *api.DataService) error {
	questionId, questionTitle, questionType, err := dataApi.GetQuestionInfo(q.QuestionTag, EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	q.QuestionId = questionId
	q.QuestionTitle = questionTitle
	q.QuestionType = questionType

	// go over the potential ansnwer tags to create potential outcome blocks
	q.PotentialOutcomes = make([]*PotentialOutcome, len(q.PotentialAnswerTags))
	for i, answerTag := range q.PotentialAnswerTags {
		outcomeId, outcome, outcomeType, err := dataApi.GetOutcomeInfo(answerTag, EN_LANGUAGE_ID)
		if err != nil {
			return err
		}

		potentialOutcome := &PotentialOutcome{OutcomeId: outcomeId,
			Outcome:     outcome,
			OutcomeType: outcomeType}
		q.PotentialOutcomes[i] = potentialOutcome
	}

	if q.ConditionBlock != nil {
		err := q.ConditionBlock.FillInDatabaseInfo(dataApi)
		if err != nil {
			return err
		}
	}

	if q.Tips != nil {
		err := q.Tips.FillInDatabaseInfo(dataApi)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Screen) FillInDatabaseInfo(dataApi *api.DataService) error {
	if s.ConditionBlock != nil {
		err := s.ConditionBlock.FillInDatabaseInfo(dataApi)
		if err != nil {
			return err
		}
	}

	if s.Questions != nil {
		for _, question := range s.Questions {
			err := question.FillInDatabaseInfo(dataApi)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Section) FillInDatabaseInfo(dataApi *api.DataService) error {
	sectionId, sectionTitle, err := dataApi.GetSectionInfo(s.SectionTag, EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	s.SectionId = sectionId
	s.SectionTitle = sectionTitle
	for _, screen := range s.Screens {
		err := screen.FillInDatabaseInfo(dataApi)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Treatment) FillInDatabaseInfo(dataApi *api.DataService) error {
	treatmentId, err := dataApi.GetTreatmentInfo(t.TreatmentTag, EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	t.TreatmentId = treatmentId
	for _, section := range t.Sections {
		err := section.FillInDatabaseInfo(dataApi)
		if err != nil {
			return err
		}
	}
	return nil
}
