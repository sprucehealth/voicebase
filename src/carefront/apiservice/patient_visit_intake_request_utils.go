package apiservice

import (
	"carefront/api"
	"carefront/info_intake"
	"encoding/json"
)

func getClientLayoutForPatientVisit(patientVisitId, languageId int64, dataApi api.DataAPI, layoutStorageService api.CloudStorageAPI) (*info_intake.InfoIntakeLayout, int64, error) {
	layoutVersionId, err := dataApi.GetLayoutVersionIdForPatientVisit(patientVisitId)
	if err != nil {
		return nil, 0, err
	}

	bucket, key, region, err := dataApi.GetStorageInfoForClientLayout(layoutVersionId, languageId)
	if err != nil {
		return nil, 0, err
	}

	patientVisitLayout, err := getHealthConditionObjectAtLocation(bucket, key, region, layoutStorageService)
	return patientVisitLayout, layoutVersionId, err
}

func getHealthConditionObjectAtLocation(bucket, key, region string, layoutStorageService api.CloudStorageAPI) (*info_intake.InfoIntakeLayout, error) {
	data, _, err := layoutStorageService.GetObjectAtLocation(bucket, key, region)
	if err != nil {
		return nil, err
	}
	healthCondition := &info_intake.InfoIntakeLayout{}
	if err := json.Unmarshal(data, healthCondition); err != nil {
		return nil, err
	}
	return healthCondition, nil
}

func getQuestionIdsInPatientVisitLayout(patientVisitLayout *info_intake.InfoIntakeLayout) []int64 {
	questionIds := make([]int64, 0)
	for _, section := range patientVisitLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				questionIds = append(questionIds, question.QuestionId)
			}
		}
	}
	return questionIds
}

func getQuestionsInPatientVisitLayout(patientVisitLayout *info_intake.InfoIntakeLayout) []*info_intake.Question {
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
