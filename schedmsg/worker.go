package schedmsg

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/patient"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/messages"
)

var (
	defaultTimePeriod = 20
)

type Worker struct {
	dataAPI      api.DataAPI
	authAPI      api.AuthAPI
	dispatcher   *dispatch.Dispatcher
	emailService email.Service
	timePeriod   int
	stopCh       chan bool
}

func StartWorker(
	dataAPI api.DataAPI, authAPI api.AuthAPI, dispatcher *dispatch.Dispatcher,
	emailService email.Service, metricsRegistry metrics.Registry, timePeriod int,
) *Worker {
	w := NewWorker(dataAPI, authAPI, dispatcher, emailService, metricsRegistry, timePeriod)
	w.Start()
	return w
}

func NewWorker(
	dataAPI api.DataAPI, authAPI api.AuthAPI, dispatcher *dispatch.Dispatcher,
	emailService email.Service, metricsRegistry metrics.Registry, timePeriod int,
) *Worker {
	tPeriod := timePeriod
	if tPeriod == 0 {
		tPeriod = defaultTimePeriod
	}
	return &Worker{
		dataAPI:      dataAPI,
		authAPI:      authAPI,
		dispatcher:   dispatcher,
		emailService: emailService,
		timePeriod:   tPeriod,
		stopCh:       make(chan bool),
	}
}

func (w *Worker) Start() {
	go func() {
		for {
			select {
			case <-w.stopCh:
				return
			default:
			}

			msgConsumed, err := w.ConsumeMessage()
			if err != nil {
				golog.Errorf(err.Error())
			}

			if !msgConsumed {
				select {
				case <-w.stopCh:
					return
				case <-time.After(time.Duration(w.timePeriod) * time.Second):
				}
			}
		}
	}()
}

func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) ConsumeMessage() (bool, error) {
	scheduledMessage, err := w.dataAPI.RandomlyPickAndStartProcessingScheduledMessage(ScheduledMsgTypes)
	if api.IsErrNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	if err := w.processMessage(scheduledMessage); err != nil {
		golog.Errorf(err.Error())
		// revert the status back to being in the scheduled state
		if err := w.dataAPI.UpdateScheduledMessage(scheduledMessage.ID, common.SMScheduled); err != nil {
			golog.Errorf(err.Error())
			return false, err
		}
		return false, err
	}

	// update the status to indicate that the message was succesfully sent
	if err := w.dataAPI.UpdateScheduledMessage(scheduledMessage.ID, common.SMSent); err != nil {
		golog.Errorf(err.Error())
		return false, err
	}

	return true, nil
}

func (w *Worker) processMessage(schedMsg *common.ScheduledMessage) error {
	// determine whether we are sending a case message or an email
	switch schedMsg.Message.TypeName() {
	case common.SMCaseMessageType:
		appMessage := schedMsg.Message.(*CaseMessage)

		patientCase, err := w.dataAPI.GetPatientCaseFromID(appMessage.PatientCaseID)
		if err != nil {
			golog.Errorf(err.Error())
			return err
		}

		people, err := w.dataAPI.GetPeople([]int64{appMessage.SenderPersonID})
		if err != nil {
			return err
		}

		msg := &common.CaseMessage{
			PersonID: appMessage.SenderPersonID,
			Body:     appMessage.Message,
			CaseID:   appMessage.PatientCaseID,
		}

		if err := messages.CreateMessageAndAttachments(msg, appMessage.Attachments,
			appMessage.SenderPersonID, appMessage.ProviderID, appMessage.SenderRole, w.dataAPI); err != nil {
			golog.Errorf(err.Error())
			return err
		}

		w.dispatcher.Publish(&messages.PostEvent{
			Message: msg,
			Case:    patientCase,
			Person:  people[appMessage.SenderPersonID],
		})

	case common.SMEmailMessageType:
		eMsg := schedMsg.Message.(*EmailMessage)
		if err := w.emailService.Send(&eMsg.Email); err != nil {
			golog.Errorf(err.Error())
			return err
		}
	case common.SMTreatmanPlanMessageType:
		sm := schedMsg.Message.(*TreatmentPlanMessage)

		// Make sure treatment plan is still active. This will happen if a treatment plan was revised after messages
		// were scheduled. It's fine. Just want to make sure not to send the messages.
		tp, err := w.dataAPI.GetAbridgedTreatmentPlan(sm.TreatmentPlanID, 0)
		if err != nil {
			return err
		}
		if tp.Status != api.STATUS_ACTIVE {
			golog.Infof("Treatmnet plan %d not active when trying to send scheduled message %d", sm.TreatmentPlanID, sm.MessageID)
			return nil
		}

		msg, err := w.dataAPI.TreatmentPlanScheduledMessage(sm.MessageID)
		if err != nil {
			return err
		}

		pcase, err := w.dataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
		if err != nil {
			return err
		}

		personID, err := w.dataAPI.GetPersonIDByRole(api.DOCTOR_ROLE, tp.DoctorID.Int64())
		if err != nil {
			return err
		}
		people, err := w.dataAPI.GetPeople([]int64{personID})
		if err != nil {
			return err
		}

		// Create follow-up visits when necessary
		for _, a := range msg.Attachments {
			if a.ItemType == common.AttachmentTypeFollowupVisit {
				pat, err := w.dataAPI.GetPatientFromID(tp.PatientID)
				if err != nil {
					return err
				}

				fvisit, err := patient.CreatePendingFollowup(pat, w.dataAPI, w.authAPI, w.dispatcher)
				if err != nil {
					return err
				}

				a.ItemID = fvisit.PatientVisitID.Int64()
				break
			}
		}

		cmsg := &common.CaseMessage{
			PersonID:    personID,
			Body:        msg.Message,
			CaseID:      pcase.ID.Int64(),
			Attachments: msg.Attachments,
		}

		msg.ID, err = w.dataAPI.CreateCaseMessage(cmsg)
		if err != nil {
			return err
		}

		w.dispatcher.Publish(&messages.PostEvent{
			Message: cmsg,
			Case:    pcase,
			Person:  people[personID],
		})
	default:
		return fmt.Errorf("Unknown message type: %s", schedMsg.Message.TypeName())
	}

	return nil
}
