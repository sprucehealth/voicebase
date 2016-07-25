package deployment

import (
	"encoding/json"
	"time"

	"context"

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

// newNotificationWorker returns a worker that consumes SQS queue at the provided URL expecting deployment notificatins
func newNotificationWorker(manager *Manager, sqs sqsiface.SQSAPI, deployNotificationQueueURL string) *notificationWorker {
	w := &notificationWorker{
		sqs:     sqs,
		manager: manager,
	}
	w.nWorker = awsutil.NewSQSWorker(sqs, deployNotificationQueueURL, w.processSQSEvent)
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

// TODO: Don't make this so Build complete specific
func (w *notificationWorker) processSQSEvent(ctx context.Context, msg string) error {
	golog.Debugf("Received SQS Message: %s", msg)
	ev := &deploy.BuildCompleteEvent{}
	if err := json.Unmarshal([]byte(msg), ev); err != nil {
		golog.Errorf("Failed to unmarshal event: %s", err)
		return nil
	}
	return w.processEvent(ctx, ev)
}

func (w *notificationWorker) processEvent(ctx context.Context, ev *deploy.BuildCompleteEvent) error {
	deploymentIDs, err := w.manager.ProcessBuildCompleteEvent(ev)
	if err != nil {
		golog.Errorf("Error while processing build complete event, discarding event: %s", err)
	} else {
		golog.Infof("Initiated deployments %v", deploymentIDs)
	}
	return nil
}
