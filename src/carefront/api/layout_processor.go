package api

import (
	"strconv"
)

func (c *Condition) FillInDatabaseInfo(dataApi DataAPI, languageId int64) error {
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
		outcomeIds, _, _, outcomeTags, _, err := dataApi.GetOutcomeInfo(questionId, languageId)
		if err != nil {
			return err
		}
		for j, outcomeTag := range outcomeTags {
			if outcomeTag == tag {
				c.PotentialAnswersId[i] = strconv.Itoa(int(outcomeIds[j]))
				break
			}
		}
	}
	return nil
}

func (t *TipSection) FillInDatabaseInfo(dataApi DataAPI, languageId int64) error {
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

func (q *Question) FillInDatabaseInfo(dataApi DataAPI, languageId int64) error {
	questionId, questionTitle, questionType, err := dataApi.GetQuestionInfo(q.QuestionTag, languageId)
	if err != nil {
		return err
	}
	q.QuestionId = questionId
	q.QuestionTitle = questionTitle
	q.QuestionType = questionType

	// go over the potential ansnwer tags to create potential outcome blocks
	q.PotentialOutcomes = make([]*PotentialOutcome, 1, 5)
	outcomeIds, outcomes, outcomeTypes, _, orderings, err := dataApi.GetOutcomeInfo(questionId, languageId)
	if err != nil {
		return err
	}

	for i, _ := range outcomeIds {
		potentialOutcome := &PotentialOutcome{OutcomeId: outcomeIds[i],
			Outcome:     outcomes[i],
			OutcomeType: outcomeTypes[i],
			Ordering:    orderings[i]}
		q.PotentialOutcomes = append(q.PotentialOutcomes, potentialOutcome)
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

func (s *Screen) FillInDatabaseInfo(dataApi DataAPI, languageId int64) error {
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

func (s *Section) FillInDatabaseInfo(dataApi DataAPI, languageId int64) error {
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

func (t *Treatment) FillInDatabaseInfo(dataApi DataAPI, languageId int64) error {
	treatmentId, err := dataApi.GetTreatmentInfo(t.TreatmentTag)
	if err != nil {
		return err
	}
	t.TreatmentId = treatmentId
	for _, section := range t.Sections {
		err := section.FillInDatabaseInfo(dataApi, languageId)
		if err != nil {
			return err
		}
	}
	return nil
}
