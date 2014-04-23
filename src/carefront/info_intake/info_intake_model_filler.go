package info_intake

import (
	"carefront/api"
	"strconv"
	"strings"
)

type HealthConditionIntakeModelProcessor struct {
	DataApi api.DataAPI
}

func (c *Condition) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	if c.QuestionTag == "" {
		return nil
	}
	questionInfo, err := dataApi.GetQuestionInfo(c.QuestionTag, languageId)
	if err != nil {
		return err
	}
	c.QuestionId = questionInfo.Id
	c.PotentialAnswersId = make([]string, len(c.PotentialAnswersTags))
	for i, tag := range c.PotentialAnswersTags {
		answerInfos, err := dataApi.GetAnswerInfo(questionInfo.Id, languageId)
		if err != nil {
			return err
		}
		for _, answerInfo := range answerInfos {
			if answerInfo.AnswerTag == tag {
				c.PotentialAnswersId[i] = strconv.Itoa(int(answerInfo.PotentialAnswerId))
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
	questionInfo, err := dataApi.GetQuestionInfo(q.QuestionTag, languageId)
	if err != nil {
		return err
	}
	q.QuestionId = questionInfo.Id
	q.QuestionTitle = questionInfo.Title
	q.QuestionTypes = []string{questionInfo.Type}
	q.ParentQuestionId = questionInfo.ParentQuestionId
	q.QuestionSummary = questionInfo.Summary
	q.AdditionalFields = questionInfo.AdditionalFields
	q.QuestionSubText = questionInfo.SubText
	q.Required = questionInfo.Required
	q.ToAlert = questionInfo.ToAlert
	q.AlertFormattedText = questionInfo.AlertFormattedText
	if questionInfo.FormattedFieldTags != "" {
		q.FormattedFieldTags = strings.Split(questionInfo.FormattedFieldTags, ",")
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

	if q.Questions != nil {
		for _, question := range q.Questions {
			err := question.FillInDatabaseInfo(dataApi, languageId)
			if err != nil {
				return err
			}
		}
	}
	// go over the potential ansnwer tags to create potential outcome blocks
	q.PotentialAnswers = make([]*PotentialAnswer, 0)
	answerInfos, err := dataApi.GetAnswerInfo(questionInfo.Id, languageId)
	if err != nil {
		return err
	}

	for _, answerInfo := range answerInfos {
		potentialAnswer := &PotentialAnswer{AnswerId: answerInfo.PotentialAnswerId,
			Answer:        answerInfo.Answer,
			AnswerSummary: answerInfo.AnswerSummary,
			AnswerTypes:   []string{answerInfo.AnswerType},
			Ordering:      answerInfo.Ordering,
			ToAlert:       answerInfo.ToAlert,
		}

		q.PotentialAnswers = append(q.PotentialAnswers, potentialAnswer)
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

func (t *InfoIntakeLayout) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
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
