package worker

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/source/hint"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/worker"
)

type initiateSync struct {
	dl                   dal.DAL
	syncEventsQueueURL   string
	initiateSyncQueueURL string
	sqsAPI               sqsiface.SQSAPI
	worker               worker.Worker
}

// NewInitiateSync returns a worker that is responsible for processing messages
// to intiate the sync process for a particular organization.
func NewInitateSync(
	dl dal.DAL,
	syncEventsQueueURL, initiateSyncQueueURL string,
	sqsAPI sqsiface.SQSAPI,
) Service {
	s := &initiateSync{
		dl:                   dl,
		syncEventsQueueURL:   syncEventsQueueURL,
		initiateSyncQueueURL: initiateSyncQueueURL,
		sqsAPI:               sqsAPI,
	}
	s.worker = awsutil.NewSQSWorker(sqsAPI, initiateSyncQueueURL, s.processInitiateSync)
	return s
}

func (s *initiateSync) Start() {
	s.worker.Start()
}

func (s *initiateSync) Shutdown() error {
	s.worker.Stop(time.Second * 30)
	return nil
}

func (s *initiateSync) processInitiateSync(ctx context.Context, data string) error {
	var initiate sync.Initiate
	if err := initiate.Unmarshal([]byte(data)); err != nil {
		return errors.Trace(err)
	}

	switch initiate.Source {
	case sync.SOURCE_HINT:
		if err := hint.DoInitialSync(s.dl, initiate.OrganizationEntityID, s.syncEventsQueueURL, s.sqsAPI); err != nil {
			return errors.Trace(err)
		}
	default:
		return errors.Errorf("Unknown source %s", initiate.Source)
	}

	return nil
}
