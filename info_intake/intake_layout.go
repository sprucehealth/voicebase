package info_intake

import (
	"github.com/sprucehealth/backend/common"
)

type InfoIntakeLayout struct {
	PathwayTag  string               `json:"health_condition"`
	PathwayID   int64                `json:"health_condition_id,string,omitempty"`
	Templated   bool                 `json:"is_templated"`
	Header      *VisitOverviewHeader `json:"visit_overview_header,omitempty"`
	Transitions []*TransitionItem    `json:"transitions,omitempty"`
	Sections    []*Section           `json:"sections"`

	// These used to be part of the intake layout but are now considered part of the container of the
	// InfoIntakeLayout instead. This is because its not the responsibility of the visit manager to care about these values
	// but more of the client to consume after the visit is complete.
	DeprecatedSKUType                *string                     `json:"cost_item_type"`
	DeprecatedAdditionalMessage      *VisitMessage               `json:"additional_message,omitempty"`
	DeprecatedSubmissionConfirmation *SubmissionConfirmationText `json:"submission_confirmation,omitempty"`
	DeprecatedCheckout               *CheckoutText               `json:"checkout,omitempty"`
}

func (i *InfoIntakeLayout) NonPhotoQuestionIDs() []int64 {
	return i.questionIDs(func(q *Question) bool {
		return q.QuestionType != QuestionTypePhotoSection
	})
}

func (i *InfoIntakeLayout) PhotoQuestionIDs() []int64 {
	return i.questionIDs(func(q *Question) bool {
		return q.QuestionType == QuestionTypePhotoSection
	})
}

func (i *InfoIntakeLayout) Questions() []*Question {
	var questions []*Question
	for _, section := range i.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				questions = append(questions, question)
			}
		}
	}
	// TODO: make sure questions is not nil. This may not be necessary
	// but to avoid obscure bugs settings to an empty slice for now.
	if len(questions) == 0 {
		questions = []*Question{}
	}
	return questions
}

func (i *InfoIntakeLayout) Answers() map[int64][]common.Answer {
	answers := make(map[int64][]common.Answer)
	for _, section := range i.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				answers[question.QuestionID] = question.Answers
			}
		}
	}
	return answers
}

func (i *InfoIntakeLayout) questionIDs(condition func(question *Question) bool) []int64 {
	var questionIDs []int64
	for _, section := range i.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if condition(question) {
					questionIDs = append(questionIDs, question.QuestionID)
				}
			}
		}
	}
	return questionIDs
}
