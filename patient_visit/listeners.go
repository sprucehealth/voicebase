package patient_visit

import (
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/schedmsg"
)

const (
	textReplacementIdentifier    = "XXX"
	insuranceCoverageQuestionTag = "q_insurance_coverage"
	noInsuranceAnswerTag         = "a_no_insurance"
	insuredPatientEvent          = "insured_patient"
	uninsuredPatientEvent        = "uninsured_patient"
)

type medAffordabilityContext struct {
	PatientFirstName         string
	ProviderShortDisplayName string
}

func init() {
	schedmsg.MustRegisterEvent(insuredPatientEvent)
	schedmsg.MustRegisterEvent(uninsuredPatientEvent)
}

func InitListeners(dataAPI api.DataAPI, apiDomain string, dispatcher *dispatch.Dispatcher, visitQueue *common.SQSQueue) {
	// Populate alerts for patient based on visit intake
	dispatcher.SubscribeAsync(func(ev *patient.VisitSubmittedEvent) error {
		processPatientAnswers(dataAPI, apiDomain, ev)
		return nil
	})
	dispatcher.Subscribe(func(ev *patient.VisitSubmittedEvent) error {
		enqueueJobToChargeAndRouteVisit(dataAPI, dispatcher, visitQueue, ev)
		return nil
	})

	// mark patient visits belonging to the case as treated if there are submitted
	// but untreated patient visits
	dispatcher.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {

		// get the list of submitted but not treated visits in the case
		visits, err := dataAPI.GetVisitsForCase(ev.TreatmentPlan.PatientCaseID.Int64(), common.SubmittedPatientVisitStates())
		if err != nil {
			golog.Errorf(err.Error())
			return err
		}

		// given that a treatment plan was acitivated, go ahead and udpate the states of these visits to indicate that
		// they were treated
		visitIDs := make([]int64, len(visits))
		for i, visit := range visits {
			visitIDs[i] = visit.PatientVisitID.Int64()
		}

		nextStatus := common.PVStatusTreated
		now := time.Now()
		if err := dataAPI.UpdatePatientVisits(visitIDs, &api.PatientVisitUpdate{
			Status:     &nextStatus,
			ClosedDate: &now,
		}); err != nil {
			golog.Errorf(err.Error())
			return err
		}

		return nil
	})
}

func enqueueJobToChargeAndRouteVisit(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, visitQueue *common.SQSQueue, ev *patient.VisitSubmittedEvent) {
	// get the active cost of the acne visit so that we can snapshot it for
	// what to charge the patient
	itemCost, err := dataAPI.GetActiveItemCost(ev.Visit.SKU)
	if api.IsErrNotFound(err) {
		// if a cost doesn't exist directly publish the charged event so that the
		// case can be routed
		dispatcher.Publish(&cost.VisitChargedEvent{
			PatientID:     ev.PatientID,
			AccountID:     ev.AccountID,
			PatientCaseID: ev.PatientCaseID,
			VisitID:       ev.VisitID,
			IsFollowup:    ev.Visit.IsFollowup,
		})
		return
	} else if err != nil {
		golog.Errorf("unable to get cost of item: %s", err)
	}

	var itemCostID int64
	if itemCost != nil {
		itemCostID = itemCost.ID
	}

	if err := apiservice.QueueUpJob(visitQueue, &cost.VisitMessage{
		PatientVisitID: ev.VisitID,
		AccountID:      ev.AccountID,
		PatientID:      ev.PatientID,
		PatientCaseID:  ev.PatientCaseID,
		ItemType:       ev.Visit.SKU,
		IsFollowup:     ev.Visit.IsFollowup,
		ItemCostID:     itemCostID,
		CardID:         ev.CardID,
	}); err != nil {
		golog.Errorf("Unable to enqueue job for charging and routing of visit: %s", err)
	}
}

func processPatientAnswers(dataAPI api.DataAPI, apiDomain string, ev *patient.VisitSubmittedEvent) {
	visitLayout, err := apiservice.GetPatientLayoutForPatientVisit(ev.Visit, api.EN_LANGUAGE_ID, dataAPI, apiDomain)
	if err != nil {
		golog.Errorf("Unable to get layout for visit: %s", err)
		return
	}

	// get the answers the patient entered for all non-photo questions
	questions := visitLayout.Questions()
	questionIDs := visitLayout.NonPhotoQuestionIDs()
	questionIdToQuestion := make(map[int64]*info_intake.Question)
	for _, question := range questions {
		questionIdToQuestion[question.QuestionID] = question
	}

	patientAnswersForQuestions, err := dataAPI.AnswersForQuestions(questionIDs, &api.PatientIntake{
		PatientID:      ev.PatientID,
		PatientVisitID: ev.VisitID})
	if err != nil {
		golog.Errorf("Unable to get patient answers for questions: %+v", patientAnswersForQuestions)
		return
	}

	alerts := make([]*common.Alert, 0)
	for questionID, answers := range patientAnswersForQuestions {
		question := questionIdToQuestion[questionID]
		toAlert := question.ToAlert
		isInsuranceQuestion := question.QuestionTag == insuranceCoverageQuestionTag

		switch {
		case toAlert:
			if alert := determineAlert(ev.PatientID, question, answers); alert != nil {
				alerts = append(alerts, alert)
			}
		case isInsuranceQuestion:

			eventType := uninsuredPatientEvent
			if isPatientInsured(question, answers) {
				eventType = insuredPatientEvent
			}

			maAssignment, err := dataAPI.GetActiveCareTeamMemberForCase(api.MA_ROLE, ev.PatientCaseID)
			if err != nil {
				golog.Infof("Unable to get ma in the care team: %s", err)
				return
			}

			patient, err := dataAPI.GetPatientFromID(ev.PatientID)
			if err != nil {
				golog.Errorf("Unable to get patient: %s", err)
				return
			}

			ma, err := dataAPI.GetDoctorFromID(maAssignment.ProviderID)
			if err != nil {
				golog.Errorf("Unable to get ma: %s", err)
				return
			}

			if err := schedmsg.ScheduleInAppMessage(
				dataAPI,
				eventType,
				&medAffordabilityContext{
					PatientFirstName:         patient.FirstName,
					ProviderShortDisplayName: ma.ShortDisplayName,
				},
				&schedmsg.CaseInfo{
					PatientID:     ev.PatientID,
					PatientCaseID: ev.PatientCaseID,
					SenderRole:    api.MA_ROLE,
					ProviderID:    ma.DoctorID.Int64(),
					PersonID:      ma.PersonID,
				},
			); err != nil {
				golog.Errorf("Unable to schedule in app message: %s", err)
				return
			}
		}
	}

	if err := dataAPI.AddAlertsForPatient(ev.PatientID, common.AlertSourcePatientVisitIntake, alerts); err != nil {
		golog.Errorf("Unable to add alerts for patient: %s", err)
		return
	}
}

func isPatientInsured(question *info_intake.Question, patientAnswers []common.Answer) bool {
	var noInsurancePotentialAnswerId int64
	// first determine the potentialAnswerId of the noInsurance choice
	for _, potentialAnswer := range question.PotentialAnswers {
		if potentialAnswer.AnswerTag == noInsuranceAnswerTag {
			noInsurancePotentialAnswerId = potentialAnswer.AnswerID
			break
		}
	}

	// now determine if the patient selected it
	for _, answer := range patientAnswers {
		a := answer.(*common.AnswerIntake)
		if a.PotentialAnswerID.Int64() == noInsurancePotentialAnswerId {
			return false
		}
	}

	return true
}

func determineAlert(patientID int64, question *info_intake.Question, patientAnswers []common.Answer) *common.Alert {
	var alertMsg string
	switch question.QuestionType {
	case info_intake.QUESTION_TYPE_AUTOCOMPLETE:

		// populate the answers to call out in the alert
		enteredAnswers := make([]string, len(patientAnswers))
		for i, answer := range patientAnswers {
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
			for _, patientAnswer := range patientAnswers {
				pAnswer := patientAnswer.(*common.AnswerIntake)
				if pAnswer.PotentialAnswerID.Int64() == potentialAnswer.AnswerID && potentialAnswer.ToAlert {
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
		return &common.Alert{
			PatientID: patientID,
			Source:    common.AlertSourcePatientVisitIntake,
			SourceID:  question.QuestionID,
			Message:   alertMsg,
			Status:    common.PAStatusActive,
		}
	}
	return nil
}
