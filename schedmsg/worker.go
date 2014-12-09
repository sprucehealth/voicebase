package schedmsg

import (
	"fmt"
	"time"

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
	dispatcher   *dispatch.Dispatcher
	emailService email.Service
	timePeriod   int
	stopCh       chan bool
}

func StartWorker(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, emailService email.Service, metricsRegistry metrics.Registry, timePeriod int) *Worker {
	w := NewWorker(dataAPI, dispatcher, emailService, metricsRegistry, timePeriod)
	w.Start()
	return w
}

func NewWorker(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, emailService email.Service, metricsRegistry metrics.Registry, timePeriod int) *Worker {
	tPeriod := timePeriod
	if tPeriod == 0 {
		tPeriod = defaultTimePeriod
	}
	return &Worker{
		dataAPI:      dataAPI,
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
	scheduledMessage, err := w.dataAPI.RandomlyPickAndStartProcessingScheduledMessage(scheduledMsgTypes)
	if err == api.NoRowsError {
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
	switch schedMsg.MessageType {
	case common.SMCaseMessageType:
		appMessage := schedMsg.MessageJSON.(*caseMessage)

		patientCase, err := w.dataAPI.GetPatientCaseFromID(appMessage.PatientCaseID)
		if err != nil {
			golog.Errorf(err.Error())
			return err
		}

		people, err := w.dataAPI.GetPeople([]int64{appMessage.SenderPersonID})

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
		eMsg := schedMsg.MessageJSON.(*emailMessage)
		if err := w.emailService.Send(&eMsg.Email); err != nil {
			golog.Errorf(err.Error())
			return err
		}
	default:
		return fmt.Errorf("Unknown message type: %s", schedMsg.MessageType)
	}

	return nil
}
