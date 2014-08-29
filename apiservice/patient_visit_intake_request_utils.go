package apiservice

import (
	"encoding/json"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
)

func GetPatientLayoutForPatientVisit(visit *common.PatientVisit, languageId int64, dataApi api.DataAPI) (*info_intake.InfoIntakeLayout, error) {
	layoutVersion, err := dataApi.GetPatientLayout(visit.LayoutVersionId.Int64(), languageId)
	if err != nil {
		return nil, err
	}

	patientVisitLayout := &info_intake.InfoIntakeLayout{}
	if err := json.Unmarshal(layoutVersion.Layout, patientVisitLayout); err != nil {
		return nil, err
	}
	return patientVisitLayout, err
}

func GetNonPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout *info_intake.InfoIntakeLayout) []int64 {
	questionIds := make([]int64, 0)
	for _, section := range patientVisitLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionType != info_intake.QUESTION_TYPE_PHOTO_SECTION {
					questionIds = append(questionIds, question.QuestionId)
				}
			}
		}
	}
	return questionIds
}

func GetPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout *info_intake.InfoIntakeLayout) []int64 {
	questionIds := make([]int64, 0)
	for _, section := range patientVisitLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionType == info_intake.QUESTION_TYPE_PHOTO_SECTION {
					questionIds = append(questionIds, question.QuestionId)
				}
			}
		}
	}
	return questionIds
}

func GetQuestionsInPatientVisitLayout(patientVisitLayout *info_intake.InfoIntakeLayout) []*info_intake.Question {
	questions := make([]*info_intake.Question, 0)
	for _, section := range patientVisitLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				questions = append(questions, question)
			}
		}
	}
	return questions
}
