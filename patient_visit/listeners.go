package patient_visit

import (
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/patient"
)

const (
	textReplacementIdentifier = "XXX"
)

func InitListeners(dataAPI api.DataAPI, visitQueue *common.SQSQueue) {

	// Populate alerts for patient based on visit intake
	dispatch.Default.Subscribe(func(ev *patient.VisitSubmittedEvent) error {
		populatePatientAlerts(dataAPI, ev)
		enqueueJobToChargeAndRouteVisit(dataAPI, visitQueue, ev)
		return nil
	})
}

func enqueueJobToChargeAndRouteVisit(dataAPI api.DataAPI, visitQueue *common.SQSQueue, ev *patient.VisitSubmittedEvent) {
	// get the active cost of the acne visit so that we can snapshot it for
	// what to charge the patient
	itemCost, err := dataAPI.GetActiveItemCost(apiservice.AcneVisit)
	if err != nil && err != api.NoRowsError {
		golog.Errorf("unable to get cost of item: %s", err)
	}

	// if a cost doesn't exist directly publish the charged event so that the
	// case can be routed
	if err == api.NoRowsError {
		dispatch.Default.Publish(&VisitChargedEvent{
			PatientID:     ev.PatientId,
			PatientCaseID: ev.PatientCaseId,
			VisitID:       ev.VisitId,
		})

		return
	}

	var itemCostId int64
	if itemCost != nil {
		itemCostId = itemCost.ID
	}

	if err := apiservice.QueueUpJob(visitQueue, &visitMessage{
		PatientVisitID: ev.VisitId,
		PatientID:      ev.PatientId,
		PatientCaseID:  ev.PatientCaseId,
		ItemType:       apiservice.AcneVisit,
		ItemCostID:     itemCostId,
	}); err != nil {
		golog.Errorf("Unable to enqueue job for charging and routing of visit: %s", err)
	}
}

func populatePatientAlerts(dataAPI api.DataAPI, ev *patient.VisitSubmittedEvent) {
	go func() {

		patientVisitLayout, err := apiservice.GetPatientLayoutForPatientVisit(ev.Visit, api.EN_LANGUAGE_ID, dataAPI)
		if err != nil {
			golog.Errorf("Unable to get layout for visit: %s", err)
			return
		}

		// get the answers the patient entered for all non-photo questions
		questions := apiservice.GetQuestionsInPatientVisitLayout(patientVisitLayout)
		questionIds := apiservice.GetNonPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)
		questionIdToQuestion := make(map[int64]*info_intake.Question)
		for _, question := range questions {
			questionIdToQuestion[question.QuestionId] = question
		}
		patientAnswersForQuestions, err := dataAPI.GetPatientAnswersForQuestions(questionIds, ev.PatientId, ev.VisitId)
		if err != nil {
			golog.Errorf("Unable to get patient answers for questions: %+v", patientAnswersForQuestions)
			return
		}

		alerts := make([]*common.Alert, 0)
		for questionId, answers := range patientAnswersForQuestions {

			// check if the alert flag is set on the question
			question := questionIdToQuestion[questionId]
			if question.ToAlert {

				var alertMsg string

				switch question.QuestionType {
				case info_intake.QUESTION_TYPE_AUTOCOMPLETE:

					// populate the answers to call out in the alert
					enteredAnswers := make([]string, len(answers))
					for i, answer := range answers {
						a := answer.(*common.AnswerIntake)
						if a.AnswerText != "" {
							enteredAnswers[i] = a.AnswerText
						} else if a.AnswerSummary != "" {
							enteredAnswers[i] = a.AnswerSummary
						} else if a.PotentialAnswer != "" {
							enteredAnswers[i] = a.PotentialAnswer
						}
					}

					alertMsg = strings.Replace(question.AlertFormattedText, textReplacementIdentifier, strings.Join(enteredAnswers, ", "), -1)

				case info_intake.QUESTION_TYPE_MULTIPLE_CHOICE, info_intake.QUESTION_TYPE_SINGLE_SELECT:
					selectedAnswers := make([]string, 0, len(question.PotentialAnswers))

					// go through all the potential answers of the question to identify the
					// ones that need to be alerted on
					for _, potentialAnswer := range question.PotentialAnswers {
						for _, patientAnswer := range answers {
							pAnswer := patientAnswer.(*common.AnswerIntake)
							if pAnswer.PotentialAnswerId.Int64() == potentialAnswer.AnswerId && potentialAnswer.ToAlert {
								if potentialAnswer.AnswerSummary != "" {
									selectedAnswers = append(selectedAnswers, potentialAnswer.AnswerSummary)
								} else {
									selectedAnswers = append(selectedAnswers, potentialAnswer.Answer)
								}
								break
							}
						}
					}

					// its possible that the patient selected an answer that need not be alerted on
					if len(selectedAnswers) > 0 {
						alertMsg = strings.Replace(question.AlertFormattedText, textReplacementIdentifier, strings.Join(selectedAnswers, ", "), -1)
					}
				}

				// TODO: Currently treating the questionId as the source for the intake,
				// but this may not scale depending on whether we get the patient to answer the same question again
				// as part of another visit.
				if alertMsg != "" {
					alerts = append(alerts, &common.Alert{
						PatientId: ev.PatientId,
						Source:    common.AlertSourcePatientVisitIntake,
						SourceId:  questionId,
						Message:   alertMsg,
					})
				}
			}
		}

		if err := dataAPI.AddAlertsForPatient(ev.PatientId, alerts); err != nil {
			golog.Errorf("Unable to add alerts for patient: %s", err)
			return
		}
	}()

}
