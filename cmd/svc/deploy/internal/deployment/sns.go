package deployment

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/deploy"
)

type notificationWorker struct {
	sqs     sqsiface.SQSAPI
	nWorker *awsutil.SQSWorker
	manager *Manager
}

type snsMessage struct {
	Message []byte
}

// newNotificationWorker returns a worker that consumes SQS queue at the provided URL expecting deployment notificatins
func newNotificationWorker(manager *Manager, sqs sqsiface.SQSAPI, deployNotificationQueueURL string) *notificationWorker {
	w := &notificationWorker{
		sqs:     sqs,
		manager: manager,
	}
	w.nWorker = awsutil.NewSQSWorker(sqs, deployNotificationQueueURL, w.processSNSEvent)
	return w
}

func (w *notificationWorker) Start() {
	w.nWorker.Start()
}

func (w *notificationWorker) Stop(wait time.Duration) {
	w.nWorker.Stop(wait)
}

func (w *notificationWorker) Started() bool {
	return w.nWorker.Started()
}

func (w *notificationWorker) processSNSEvent(msg string) error {
	var snsMsg snsMessage
	if err := json.Unmarshal([]byte(msg), &snsMsg); err != nil {
		golog.Errorf("Failed to unmarshal sns message: %s", err.Error())
		return nil
	}
	ev := &deploy.Envelope{}
	if err := json.Unmarshal(snsMsg.Message, ev); err != nil {
		golog.Errorf("Failed to unmarshal event envelope: %s", err)
		return nil
	}
	return w.processEvent(ev.Event)
}

func (w *notificationWorker) processEvent(ev deploy.Event) error {
	switch ev.Type() {
	case deploy.BuildComplete:
		deploymentIDs, err := w.manager.ProcessBuildCompleteEvent(ev.(*deploy.BuildCompleteEvent))
		if err != nil {
			golog.Errorf("Error while processing build complete event, discarding event: %s", err)
		} else {
			golog.Infof("Initiated deployments %v", deploymentIDs)
		}
		return nil
	default:
		golog.Errorf("Unknown event type %s ignoring", ev.Type())
	}
	return nil
}
