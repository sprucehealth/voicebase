package schedmsg

import (
	"encoding/json"
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
	queue        *common.SQSQueue
	emailService email.Service
}

func StartWorker(dataAPI api.DataAPI, queue *common.SQSQueue,
	emailService email.Service, metricsRegistry metrics.Registry) {
	(&worker{
		dataAPI:      dataAPI,
		queue:        queue,
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
	msgs, err := w.queue.QueueService.ReceiveMessage(w.queue.QueueUrl, nil, batchSize, visibilityTimeout, waitTimeSeconds)
	if err != nil {
		return false, err
	}

	allMsgsConsumed := len(msgs) > 0

	for _, m := range msgs {
		v := &schedSQSMessage{}
		if err := json.Unmarshal([]byte(m.Body), v); err != nil {
			return false, err
		}

		// nothing to do with a scheduled message that has not reached its threshold yet
		if v.ScheduledTime.Before(time.Now()) {
			allMsgsConsumed = false
			continue
		}

		if err := w.processMessage(v); err != nil {
			golog.Errorf(err.Error())
			allMsgsConsumed = false
		} else {
			if err := w.queue.QueueService.DeleteMessage(w.queue.QueueUrl, m.ReceiptHandle); err != nil {
				golog.Errorf(err.Error())
				allMsgsConsumed = false
			}
		}
	}

	return allMsgsConsumed, nil
}

func (w *worker) processMessage(m *schedSQSMessage) error {
	schedMsg, err := w.dataAPI.ScheduledMessage(m.ScheduledMessageID, scheduledMsgTypes)
	if err != nil {
		return err
	}

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
