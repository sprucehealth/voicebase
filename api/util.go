package api

import (
	"database/sql"
	"fmt"
	"github.com/sprucehealth/backend/info_intake"
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

func fillConditionBlock(c *info_intake.Condition, dataApi DataAPI, languageId int64) error {
	if c.QuestionTag == "" {
		return nil
	}
	questionInfo, err := dataApi.GetQuestionInfo(c.QuestionTag, languageId)
	if err != nil {
		return err
	}
	c.QuestionId = questionInfo.QuestionId
	c.PotentialAnswersId = make([]string, len(c.PotentialAnswersTags))
	for i, tag := range c.PotentialAnswersTags {
		answerInfos, err := dataApi.GetAnswerInfo(questionInfo.QuestionId, languageId)
		if err != nil {
			return err
		}
		for _, answerInfo := range answerInfos {
			if answerInfo.AnswerTag == tag {
				c.PotentialAnswersId[i] = strconv.Itoa(int(answerInfo.AnswerId))
				break
			}
		}
		if c.PotentialAnswersId[i] == "" {
			return fmt.Errorf("Unknown answer tag '%s' for question '%s'", tag, c.QuestionTag)
		}
	}
	return nil
}

func fillTipSection(t *info_intake.TipSection, dataApi DataAPI, languageId int64) error {
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

func fillQuestion(q *info_intake.Question, dataApi DataAPI, languageId int64) error {
	questionInfo, err := dataApi.GetQuestionInfo(q.QuestionTag, languageId)
	if err == NoRowsError {
		return fmt.Errorf("no question with tag '%s'", q.QuestionTag)
	} else if err != nil {
		return err
	}
	q.QuestionId = questionInfo.QuestionId
	q.QuestionTitle = questionInfo.QuestionTitle
	q.QuestionType = questionInfo.QuestionType
	q.ParentQuestionId = questionInfo.ParentQuestionId
	q.QuestionSummary = questionInfo.QuestionSummary
	q.AdditionalFields = questionInfo.AdditionalFields
	q.QuestionSubText = questionInfo.QuestionSubText
	q.Required = questionInfo.Required
	q.ToAlert = questionInfo.ToAlert
	q.QuestionTitleHasTokens = questionInfo.QuestionTitleHasTokens
	q.AlertFormattedText = questionInfo.AlertFormattedText
	if questionInfo.FormattedFieldTags != nil {
		q.FormattedFieldTags = strings.Split(questionInfo.FormattedFieldTags[0], ",")
	}

	if q.ConditionBlock != nil {
		err := fillConditionBlock(q.ConditionBlock, dataApi, languageId)
		if err != nil {
			return err
		}
	}

	if q.Tips != nil {
		err := fillTipSection(q.Tips, dataApi, languageId)
		if err != nil {
			return err
		}
	}

	// the subquestion config can specify either a list of screens and/or questions
	if q.SubQuestionsConfig != nil {
		if q.SubQuestionsConfig.Questions != nil {
			for _, question := range q.SubQuestionsConfig.Questions {
				if err := fillQuestion(question, dataApi, languageId); err != nil {
					return err
				}
			}
		}

		if q.SubQuestionsConfig.Screens != nil {
			for _, screen := range q.SubQuestionsConfig.Screens {
				if err := fillScreen(screen, dataApi, languageId); err != nil {
					return err
				}
			}
		}
	}

	// go over the potential ansnwer tags to create potential outcome blocks
	q.PotentialAnswers, err = dataApi.GetAnswerInfo(questionInfo.QuestionId, languageId)
	if err != nil {
		return err
	}

	// fill in any photo slots into the question
	// Note that this could be optimized to only query based on the question type
	// but given the small number of questions currently coupled with the fact that we need to rewrite the implementation
	// to better organize the structure in the future its not worth to base this off the question types currently
	q.PhotoSlots, err = dataApi.GetPhotoSlots(questionInfo.QuestionId, languageId)
	if err != nil {
		return err
	}

	return nil
}

func fillScreen(s *info_intake.Screen, dataApi DataAPI, languageId int64) error {
	if s.ConditionBlock != nil {
		err := fillConditionBlock(s.ConditionBlock, dataApi, languageId)
		if err != nil {
			return err
		}
	}

	if s.Questions != nil {
		for _, question := range s.Questions {
			err := fillQuestion(question, dataApi, languageId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func fillSection(s *info_intake.Section, dataApi DataAPI, languageId int64) error {
	sectionId, sectionTitle, err := dataApi.GetSectionInfo(s.SectionTag, languageId)
	if err == NoRowsError {
		return fmt.Errorf("no section with tag '%s'", s.SectionTag)
	} else if err != nil {
		return err
	}
	s.SectionId = sectionId
	s.SectionTitle = sectionTitle
	for _, screen := range s.Screens {
		err := fillScreen(screen, dataApi, languageId)
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
		err := fillSection(section, dataApi, languageId)
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
			err := fillQuestion(question, dataApi, languageId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
