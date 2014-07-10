package patient_visit

import (
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	textReplacementIdentifier = "XXX"
)

func InitListeners(dataAPI api.DataAPI) {

	// Pull out any alerts for the patient based on the patient visit intake
	dispatch.Default.Subscribe(func(ev *VisitSubmittedEvent) error {
		go func() {

			patientVisitLayout, _, err := apiservice.GetPatientLayoutForPatientVisit(ev.VisitId, api.EN_LANGUAGE_ID, dataAPI)
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
				golog.Errorf("Unable to get patient answers for questions: %s", patientAnswersForQuestions)
				return
			}

			alerts := make([]*common.Alert, 0)
			for questionId, answers := range patientAnswersForQuestions {

				// check if the alert flag is set on the question
				question := questionIdToQuestion[questionId]

				if question.ToAlert {

					switch question.QuestionType {
					case info_intake.QUESTION_TYPE_AUTOCOMPLETE:
						enteredAnswers := make([]string, len(answers))
						// populate the answers to call out in the alert
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

						if len(answers) > 0 {
							alerts = append(alerts, &common.Alert{
								PatientId: ev.PatientId,
								Source:    common.AlertSourcePatientVisitIntake,
								SourceId:  questionId,
								Message: strings.Replace(question.AlertFormattedText,
									textReplacementIdentifier, strings.Join(enteredAnswers, ", "), -1),
							})
						}

					case info_intake.QUESTION_TYPE_MULTIPLE_CHOICE, info_intake.QUESTION_TYPE_SINGLE_SELECT:
						selectedAnswers := make([]string, 0, len(question.PotentialAnswers))
						for _, potentialAnswer := range question.PotentialAnswers {
							for _, patientAnswer := range answers {
								pAnswer := patientAnswer.(*common.AnswerIntake)
								// populate all the selected answers to show in the alert
								if pAnswer.PotentialAnswerId.Int64() == potentialAnswer.AnswerId {
									if potentialAnswer.ToAlert {
										if potentialAnswer.AnswerSummary != "" {
											selectedAnswers = append(selectedAnswers, potentialAnswer.AnswerSummary)
										} else {
											selectedAnswers = append(selectedAnswers, potentialAnswer.Answer)
										}
										break
									}
								}
							}
						}

						if len(selectedAnswers) > 0 {
							alerts = append(alerts, &common.Alert{
								PatientId: ev.PatientId,
								Source:    common.AlertSourcePatientVisitIntake,
								SourceId:  questionId,
								Message: strings.Replace(question.AlertFormattedText,
									textReplacementIdentifier, strings.Join(selectedAnswers, ", "), -1),
							})
						}
					}
				}
			}

			if err := dataAPI.AddAlertsForPatient(ev.PatientId, alerts); err != nil {
				golog.Errorf("Unable to add alerts for patient: %s", err)
			}
		}()
		return nil
	})
}
