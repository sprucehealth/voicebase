package workers

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/threading"
)

// Subscriber wraps the events subscriber with access to the DAL and clients
type Subscriber struct {
	events.Subscriber
	dal             dal.DAL
	directoryClient directory.DirectoryClient
	threadClient    SubscriptionsThreadClient
}

// SubscriptionsThreadClient represents a client that is consumed by the subscriptions workers
type SubscriptionsThreadClient interface {
	newPatientWelcomeMessageThreadClient
}

// InitSubscriptions bootstraps the PubSub subscriptions for the service
func InitSubscriptions(
	dal dal.DAL,
	directoryClient directory.DirectoryClient,
	threadClient SubscriptionsThreadClient,
	sqsAPI sqsiface.SQSAPI,
	snsAPI snsiface.SNSAPI,
	awsSession *session.Session,
	serviceName string) (events.Subscriber, error) {
	subscriber, err := events.NewSQSSubscriber(sqsAPI, snsAPI, awsSession, serviceName)
	if err != nil {
		return nil, errors.Trace(err)
	}
	s := &Subscriber{
		Subscriber:      subscriber,
		dal:             dal,
		directoryClient: directoryClient,
		threadClient:    threadClient,
	}
	if err := s.Subscribe("newpatient-welcome-message", []events.Unmarshaler{&threading.NewThreadEvent{}}, s.newPatientWelcomeMessage); err != nil {
		return nil, errors.Trace(err)
	}
	return s, nil
}
