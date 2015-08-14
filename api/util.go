package api

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
)

// db can be used when a function can accept either a *Tx or *DB.
type db interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
}

type scannable interface {
	Scan(dest ...interface{}) error
}

type NullableTime struct {
	Time  *time.Time
	Valid bool
}

func fillConditionBlock(c *info_intake.Condition, dataAPI DataAPI, languageID int64) error {
	for _, operand := range c.Operands {
		if err := fillConditionBlock(operand, dataAPI, languageID); err != nil {
			return err
		}
	}

	c.Type = c.OperationTag

	if c.QuestionTag == "" {
		return nil
	}

	// Get the latest version of this question
	version, err := dataAPI.MaxQuestionVersion(c.QuestionTag, languageID)
	if err != nil {
		return err
	}

	questionInfo, err := dataAPI.GetQuestionInfo(c.QuestionTag, languageID, version)
	if err != nil {
		return err
	}
	c.QuestionID = questionInfo.QuestionID
	c.PotentialAnswersID = make([]string, len(c.PotentialAnswersTags))
	for i, tag := range c.PotentialAnswersTags {
		answerInfos, err := dataAPI.GetAnswerInfo(questionInfo.QuestionID, languageID)
		if err != nil {
			return err
		}
		for _, answerInfo := range answerInfos {
			if answerInfo.AnswerTag == tag {
				c.PotentialAnswersID[i] = strconv.Itoa(int(answerInfo.AnswerID))
				break
			}
		}
		if c.PotentialAnswersID[i] == "" {
			return fmt.Errorf("Unknown answer tag '%s' for question '%s'", tag, c.QuestionTag)
		}
	}

	return nil
}

func fillQuestion(q *info_intake.Question, dataAPI DataAPI, languageID int64) error {
	// Get the latest version of this question
	version, err := dataAPI.MaxQuestionVersion(q.QuestionTag, languageID)
	if err != nil {
		return err
	}

	questionInfo, err := dataAPI.GetQuestionInfo(q.QuestionTag, languageID, version)
	if IsErrNotFound(err) {
		return fmt.Errorf("no question with tag '%s'", q.QuestionTag)
	} else if err != nil {
		return err
	}
	q.QuestionID = questionInfo.QuestionID
	q.QuestionTitle = questionInfo.QuestionTitle
	q.QuestionType = questionInfo.QuestionType
	q.Type = q.QuestionType
	q.ParentQuestionID = questionInfo.ParentQuestionID
	q.QuestionSummary = questionInfo.QuestionSummary
	q.QuestionSubText = questionInfo.QuestionSubText
	q.Required = questionInfo.Required
	q.ToAlert = questionInfo.ToAlert
	q.QuestionTitleHasTokens = questionInfo.QuestionTitleHasTokens
	q.AlertFormattedText = questionInfo.AlertFormattedText

	if len(q.AdditionalFields) > 0 {
		for fieldName, fieldValue := range questionInfo.AdditionalFields {
			q.AdditionalFields[fieldName] = fieldValue
		}
	} else {
		q.AdditionalFields = questionInfo.AdditionalFields
	}

	if questionInfo.FormattedFieldTags != nil {
		q.FormattedFieldTags = strings.Split(questionInfo.FormattedFieldTags[0], ",")
	}

	if q.ConditionBlock != nil {
		err := fillConditionBlock(q.ConditionBlock, dataAPI, languageID)
		if err != nil {
			return err
		}
	}

	// the subquestion config can specify either a list of screens and/or questions
	if q.SubQuestionsConfig != nil {
		if q.SubQuestionsConfig.Questions != nil {
			for _, question := range q.SubQuestionsConfig.Questions {
				if err := fillQuestion(question, dataAPI, languageID); err != nil {
					return err
				}
			}
		}

		if q.SubQuestionsConfig.Screens != nil {
			for _, screen := range q.SubQuestionsConfig.Screens {
				if err := fillScreen(screen, dataAPI, languageID); err != nil {
					return err
				}
			}
		}
	}

	// go over the potential ansnwer tags to create potential outcome blocks
	q.PotentialAnswers, err = dataAPI.GetAnswerInfo(questionInfo.QuestionID, languageID)
	if err != nil {
		return err
	}

	for i := range q.PotentialAnswers {
		q.PotentialAnswers[i].Type = q.PotentialAnswers[i].AnswerType
	}

	// fill in any photo slots into the question
	// Note that this could be optimized to only query based on the question type
	// but given the small number of questions currently coupled with the fact that we need to rewrite the implementation
	// to better organize the structure in the future its not worth to base this off the question types currently
	q.PhotoSlots, err = dataAPI.GetPhotoSlotsInfo(questionInfo.QuestionID, languageID)
	if err != nil {
		return err
	}

	return nil
}

func fillScreen(s *info_intake.Screen, dataAPI DataAPI, languageID int64) error {
	if s.ConditionBlock != nil {
		err := fillConditionBlock(s.ConditionBlock, dataAPI, languageID)
		if err != nil {
			return err
		}
	}

	if s.Questions != nil {
		for _, question := range s.Questions {
			err := fillQuestion(question, dataAPI, languageID)
			if err != nil {
				return err
			}
		}
	}

	// assume for now that if the screen type is not defined that it is a screen
	// containing questions
	s.Type = "screen_type_questions"
	if s.ScreenType != "" {
		s.Type = s.ScreenType
	}

	return nil
}

func fillSection(s *info_intake.Section, dataAPI DataAPI, languageID int64) error {
	// only attempt to get the section from the database if the layout doesn't fully describe
	// the section information
	if s.SectionTitle == "" || s.SectionID == "" {
		sectionID, sectionTitle, err := dataAPI.GetSectionInfo(s.SectionTag, languageID)
		if IsErrNotFound(err) {
			return fmt.Errorf("no section with tag '%s'", s.SectionTag)
		} else if err != nil {
			return err
		}
		s.SectionID = strconv.FormatInt(sectionID, 10)
		s.SectionTitle = sectionTitle
	}
	for _, screen := range s.Screens {
		err := fillScreen(screen, dataAPI, languageID)
		if err != nil {
			return err
		}
	}

	s.Type = "section_type_screen_container"
	return nil
}

func FillIntakeLayout(t *info_intake.InfoIntakeLayout, dataAPI DataAPI, languageID int64) error {
	pathway, err := dataAPI.PathwayForTag(t.PathwayTag, PONone)
	if err != nil {
		return err
	}
	t.PathwayID = pathway.ID
	for _, section := range t.Sections {
		err := fillSection(section, dataAPI, languageID)
		if err != nil {
			return err
		}
	}
	return nil
}

func FillDiagnosisIntake(d *info_intake.DiagnosisIntake, dataAPI DataAPI, languageID int64) error {
	// fill in the questions from the database
	for _, section := range d.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {
			err := fillQuestion(question, dataAPI, languageID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func FillQuestions(questions []*info_intake.Question, dataAPI DataAPI, languageID int64) error {
	for _, question := range questions {
		if err := fillQuestion(question, dataAPI, languageID); err != nil {
			return err
		}
	}
	return nil
}

func accountIDForPatient(db db, patientID common.PatientID) (int64, error) {
	var accountID int64
	err := db.QueryRow(`SELECT account_id FROM patient WHERE id = ?`, patientID).Scan(&accountID)
	if err == sql.ErrNoRows {
		err = ErrNotFound("patient")
	}
	return accountID, err
}
