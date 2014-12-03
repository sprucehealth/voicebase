package info_intake

import (
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/sku"
)

type InfoIntakeLayout struct {
	HealthConditionTag     string                      `json:"health_condition"`
	HealthConditionId      int64                       `json:"health_condition_id,string,omitempty"`
	Templated              bool                        `json:"is_templated"`
	SKU                    *sku.SKU                    `json:"cost_item_type"`
	Header                 *VisitOverviewHeader        `json:"visit_overview_header,omitempty"`
	AdditionalMessage      *VisitMessage               `json:"additional_message,omitempty"`
	SubmissionConfirmation *SubmissionConfirmationText `json:"submission_confirmation,omitempty"`
	Checkout               *CheckoutText               `json:"checkout,omitempty"`
	Transitions            []*TransitionItem           `json:"transitions,omitempty"`
	Sections               []*Section                  `json:"sections"`
}

func (i *InfoIntakeLayout) NonPhotoQuestionIDs() []int64 {
	return i.questionIDs(func(questionType string) bool {
		return questionType != QUESTION_TYPE_PHOTO_SECTION
	})
}

func (i *InfoIntakeLayout) PhotoQuestionIDs() []int64 {
	return i.questionIDs(func(questionType string) bool {
		return questionType == QUESTION_TYPE_PHOTO_SECTION
	})
}

func (i *InfoIntakeLayout) Questions() []*Question {
	questions := make([]*Question, 0)
	for _, section := range i.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				questions = append(questions, question)
			}
		}
	}
	return questions
}

func (i *InfoIntakeLayout) Answers() map[int64][]common.Answer {
	answers := make(map[int64][]common.Answer)
	for _, section := range i.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				answers[question.QuestionId] = question.Answers
			}
		}
	}
	return answers
}

func (i *InfoIntakeLayout) questionIDs(condition func(questionType string) bool) []int64 {
	var questionIDs []int64
	for _, section := range i.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if condition(question.QuestionType) {
					questionIDs = append(questionIDs, question.QuestionId)
				}
			}
		}
	}
	return questionIDs
}
