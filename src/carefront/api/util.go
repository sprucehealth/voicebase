package api

import (
	"carefront/info_intake"
	"database/sql"
	"strconv"
	"strings"
)

// The db interface can be used when a method can accept either
// a *Tx or *DB.
type db interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func FillConditionBlock(c *info_intake.Condition, dataApi DataAPI, languageId int64) error {
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
				c.PotentialAnswersId[i] = strconv.Itoa(int(answerInfo.AnswerId))
				break
			}
		}
	}
	return nil
}

func FillTipSection(t *info_intake.TipSection, dataApi DataAPI, languageId int64) error {
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

func FillQuestion(q *info_intake.Question, dataApi DataAPI, languageId int64) error {
	questionInfo, err := dataApi.GetQuestionInfo(q.QuestionTag, languageId)
	if err != nil {
		return err
	}
	q.QuestionId = questionInfo.Id
	q.QuestionTitle = questionInfo.Title
	q.QuestionType = questionInfo.Type
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
		err := FillConditionBlock(q.ConditionBlock, dataApi, languageId)
		if err != nil {
			return err
		}
	}

	if q.Tips != nil {
		err := FillTipSection(q.Tips, dataApi, languageId)
		if err != nil {
			return err
		}
	}

	if q.Questions != nil {
		for _, question := range q.Questions {
			err := FillQuestion(question, dataApi, languageId)
			if err != nil {
				return err
			}
		}
	}
	// go over the potential ansnwer tags to create potential outcome blocks
	q.PotentialAnswers, err = dataApi.GetAnswerInfo(questionInfo.Id, languageId)
	if err != nil {
		return err
	}

	// fill in any photo slots into the question
	// Note that this could be optimized to only query based on the question type
	// but given the small number of questions currently coupled with the fact that we need to rewrite the implementation
	// to better organize the structure in the future its not worth to base this off the question types currently
	q.PhotoSlots, err = dataApi.GetPhotoSlots(questionInfo.Id, languageId)
	if err != nil {
		return err
	}

	return nil
}

func FillScreen(s *info_intake.Screen, dataApi DataAPI, languageId int64) error {
	if s.ConditionBlock != nil {
		err := FillConditionBlock(s.ConditionBlock, dataApi, languageId)
		if err != nil {
			return err
		}
	}

	if s.Questions != nil {
		for _, question := range s.Questions {
			err := FillQuestion(question, dataApi, languageId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func FillSection(s *info_intake.Section, dataApi DataAPI, languageId int64) error {
	sectionId, sectionTitle, err := dataApi.GetSectionInfo(s.SectionTag, languageId)
	if err != nil {
		return err
	}
	s.SectionId = sectionId
	s.SectionTitle = sectionTitle
	for _, screen := range s.Screens {
		err := FillScreen(screen, dataApi, languageId)
		if err != nil {
			return err
		}
	}
	return nil
}

func FillIntakeLayout(t *info_intake.InfoIntakeLayout, dataApi DataAPI, languageId int64) error {
	healthConditionId, err := dataApi.GetHealthConditionInfo(t.HealthConditionTag)
	if err != nil {
		return err
	}
	t.HealthConditionId = healthConditionId
	for _, section := range t.Sections {
		err := FillSection(section, dataApi, languageId)
		if err != nil {
			return err
		}
	}
	return nil
}

func FillDiagnosisIntake(d *info_intake.DiagnosisIntake, dataApi DataAPI, languageId int64) error {
	// fill in the questions from the database
	for _, section := range d.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {
			err := FillQuestion(question, dataApi, languageId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
