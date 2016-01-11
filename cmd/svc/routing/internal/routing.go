package internal

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/cmd/svc/routing/internal/worker/appmsg"
	"github.com/sprucehealth/backend/cmd/svc/routing/internal/worker/externalmsg"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
)

type routingService struct {
	workers []worker.Worker
}

type RoutingService interface {
	Start()
}

func NewRoutingService(
	awsSession *session.Session,
	externalMessageQueueName, inAppMessageQueueName string,
	directory directory.DirectoryClient,
	threading threading.ThreadsClient,
	excomms excomms.ExCommsClient) (RoutingService, error) {

	externalMessageQueue := sqs.New(awsSession)
	res, err := externalMessageQueue.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: ptr.String(externalMessageQueueName),
	})
	if err != nil {
		panic(err)
	}
	externalMessageQueueURL := *res.QueueUrl

	appMessageQueue := sqs.New(awsSession)
	res, err = appMessageQueue.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: ptr.String(inAppMessageQueueName),
	})
	if err != nil {
		panic(err)
	}
	appMessageQueueURL := *res.QueueUrl

	rs := &routingService{
		workers: []worker.Worker{
			externalmsg.NewWorker(
				externalMessageQueue,
				externalMessageQueueURL,
				directory,
				threading,
			),
			appmsg.NewWorker(
				appMessageQueue,
				appMessageQueueURL,
				directory,
				excomms,
			),
		},
	}

	return rs, nil
}

func (r *routingService) Start() {
	for _, w := range r.workers {
		if w.Started() {
			continue
		}
		w.Start()
	}
}
