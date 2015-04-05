package apiservice

import (
	"errors"
	"net/http"

	"github.com/sprucehealth/backend/common"
)

type AnswerItem struct {
	PotentialAnswerID int64                 `json:"potential_answer_id,string"`
	AnswerText        string                `json:"answer_text"`
	SubQuestions      []*QuestionAnswerItem `json:"answers,omitempty"`
}

type QuestionAnswerItem struct {
	QuestionID    int64         `json:"question_id,string"`
	AnswerIntakes []*AnswerItem `json:"potential_answers"`
}

type IntakeData struct {
	PatientVisitID int64                 `json:"patient_visit_id,string"`
	SessionID      string                `json:"session_id"`
	SessionCounter uint                  `json:"counter"`
	Questions      []*QuestionAnswerItem `json:"questions"`
}

type AnswerIntakeResponse struct {
	Result string `json:"result"`
}

func (a *IntakeData) Validate(w http.ResponseWriter) error {
	if a.PatientVisitID == 0 {
		return errors.New("patient_visit_id missing")
	}

	if a.Questions == nil || len(a.Questions) == 0 {
		return errors.New("missing patient information to save for patient visit")
	}

	for _, questionItem := range a.Questions {
		if questionItem.QuestionID == 0 {
			return errors.New("question_id missing")
		}

		if questionItem.AnswerIntakes == nil {
			return errors.New("potential_answers missing")
		}
	}

	return nil
}

func TransformAnswers(answersToQuestions map[int64][]common.Answer) []*QuestionAnswerItem {

	if len(answersToQuestions) == 0 {
		return nil
	}

	answerItems := make([]*QuestionAnswerItem, len(answersToQuestions))

	var i int
	for questionID, answers := range answersToQuestions {
		answerItem := &QuestionAnswerItem{
			QuestionID:    questionID,
			AnswerIntakes: make([]*AnswerItem, len(answers)),
		}
		answerItems[i] = answerItem
		i++

		for j, answer := range answers {
			aIntake, ok := answer.(*common.AnswerIntake)

			// only work with answers that are of expected type
			if !ok {
				continue
			}

			answerItem.AnswerIntakes[j] = &AnswerItem{
				PotentialAnswerID: aIntake.PotentialAnswerID.Int64(),
				AnswerText:        aIntake.AnswerText,
				SubQuestions:      make([]*QuestionAnswerItem, len(aIntake.SubAnswers)),
			}

			// currently, we only support subquestions having a single answer item
			for k, subAnswer := range aIntake.SubAnswers {
				answerItem.AnswerIntakes[j].SubQuestions[k] = &QuestionAnswerItem{
					QuestionID: subAnswer.QuestionID.Int64(),
					AnswerIntakes: []*AnswerItem{
						&AnswerItem{
							PotentialAnswerID: subAnswer.PotentialAnswerID.Int64(),
							AnswerText:        subAnswer.AnswerText,
						},
					},
				}
			}
		}
	}

	return answerItems
}

func (a *IntakeData) Equals(other *IntakeData) bool {
	if a == nil || other == nil {
		return false
	}

	if a.PatientVisitID != other.PatientVisitID {
		return false
	}

	if len(a.Questions) != len(other.Questions) {
		return false
	}

	for i, question := range a.Questions {
		if !question.Equals(other.Questions[i]) {
			return false
		}
	}

	return true
}

func (q *QuestionAnswerItem) Equals(other *QuestionAnswerItem) bool {
	if q == nil || other == nil {
		return false
	}

	if q.QuestionID != other.QuestionID {
		return false
	}

	if len(q.AnswerIntakes) != len(other.AnswerIntakes) {
		return false
	}

	for i, answerIntake := range q.AnswerIntakes {
		if answerIntake.AnswerText != other.AnswerIntakes[i].AnswerText {
			return false
		}

		if answerIntake.PotentialAnswerID != other.AnswerIntakes[i].PotentialAnswerID {
			return false
		}

		if len(answerIntake.SubQuestions) != len(other.AnswerIntakes[i].SubQuestions) {
			return false
		}

		for j, subQuestion := range answerIntake.SubQuestions {
			if !subQuestion.Equals(other.AnswerIntakes[i].SubQuestions[j]) {
				return false
			}
		}
	}

	return true
}
