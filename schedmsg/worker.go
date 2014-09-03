package schedmsg

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

var (
	batchSize          = 1
	visibilityTimeout  = 60 * 5
	waitTimeSeconds    = 20
	timeBetweenRetries = 10
	defaultTimePeriod  = 2
)

type worker struct {
	dataAPI      api.DataAPI
	emailService email.Service
}

func StartWorker(dataAPI api.DataAPI, emailService email.Service, metricsRegistry metrics.Registry) {
	(&worker{
		dataAPI:      dataAPI,
		emailService: emailService,
	}).start()
}

func (w *worker) start() {
	go func() {
		for {
			if msgConsumed, err := w.consumeMessage(); err != nil {
				golog.Errorf(err.Error())
			} else if !msgConsumed {
				time.Sleep(time.Duration(defaultTimePeriod) * time.Second)
			}
		}
	}()
}

func (w *worker) consumeMessage() (bool, error) {

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

func (w *worker) processMessage(schedMsg *common.ScheduledMessage) error {

	// determine whether we are sending a case message or an email
	switch schedMsg.MessageType {
	case common.SMCaseMessageType:
		appMessage := schedMsg.MessageJSON.(*caseMessage)

		patientCase, err := w.dataAPI.GetPatientCaseFromId(appMessage.PatientCaseID)
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

		dispatch.Default.Publish(&messages.PostEvent{
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
