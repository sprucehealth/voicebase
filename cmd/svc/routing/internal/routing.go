package internal

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/cmd/svc/routing/internal/worker/appmsg"
	"github.com/sprucehealth/backend/cmd/svc/routing/internal/worker/externalmsg"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
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
	excomms excomms.ExCommsClient,
	settings settings.SettingsClient,
	sns snsiface.SNSAPI,
	blockAccountsTopicARN string,
	kmsKeyARN string) (RoutingService, error) {

	externalMessageQueue, err := awsutil.NewEncryptedSQS(kmsKeyARN, kms.New(awsSession), sqs.New(awsSession))
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize enrypted sqs: %s", err.Error())
	}
	res, err := externalMessageQueue.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: ptr.String(externalMessageQueueName),
	})
	if err != nil {
		panic(err)
	}
	externalMessageQueueURL := *res.QueueUrl

	appMessageQueue, err := awsutil.NewEncryptedSQS(kmsKeyARN, kms.New(awsSession), sqs.New(awsSession))
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize enrypted sqs: %s", err.Error())
	}
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
				sns,
				blockAccountsTopicARN,
				directory,
				threading,
			),
			appmsg.NewWorker(
				appMessageQueue,
				appMessageQueueURL,
				directory,
				excomms,
				settings,
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
