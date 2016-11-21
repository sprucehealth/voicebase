package events

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/worker"
)

// Unmarshaler is implemented by protocol buffer structs
type Unmarshaler interface {
	Unmarshal(data []byte) error
}

// Subscriber is implemented by any object that supports subcribing to events published to an
// SNS topic
type Subscriber interface {
	Subscribe(string, []Unmarshaler, func(u Unmarshaler) error)
	Stop()
}

type sqsSubscriber struct {
	sqsAPI        sqsiface.SQSAPI
	snsAPI        snsiface.SNSAPI
	subscriptions []*subscription
	workers       map[worker.Worker]struct{}
	serviceName   string
	sqsURLPrefix  string
}

// NewSQSSubscriber returns a subscriber that subscribes to messages published to an SQS queue
func NewSQSSubscriber(sqsAPI sqsiface.SQSAPI, snsAPI snsiface.SNSAPI, awsSession *session.Session, serviceName string) (Subscriber, error) {
	accountID, err := getAWSAccountID(awsSession)
	if err != nil {
		return nil, errors.Trace(err)
	}

	sqsURLPrefix := fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/", *awsSession.Config.Region, accountID)

	return &sqsSubscriber{
		sqsAPI:        sqsAPI,
		snsAPI:        snsAPI,
		subscriptions: make([]*subscription, 0),
		serviceName:   serviceName,
		sqsURLPrefix:  sqsURLPrefix,
	}, nil
}

// Subscribe creates a new SQS worker that receives messages from
// an SQS queue for the list of events that are provided.
// TODO: The assumption here is that the SQS queue, its subscription to a SNS topics of the name {env}-{publishingService}-{eventName}
// for all the listed events and the SNS topic itself exist. Add code here to programmatically create the SQS queue, the SNS topic and the subscription.
func (s *sqsSubscriber) Subscribe(name string, events []Unmarshaler, fn func(u Unmarshaler) error) {

	eventTypes := make(map[string]reflect.Type, len(events))
	for _, event := range events {
		eventTypes[resourceNameFromEvent(event)] = reflect.TypeOf(event)
	}

	sub := &subscription{
		fn:         fn,
		eventTypes: eventTypes,
	}
	sub.worker = awsutil.NewSQSWorker(s.sqsAPI, s.sqsURLPrefix+name, sub.processMessage)
	sub.worker.Start()

	s.subscriptions = append(s.subscriptions, sub)
}

func (s *sqsSubscriber) Stop() {
	for _, subscription := range s.subscriptions {
		subscription.worker.Stop(time.Second * 30)
	}
}

type subscription struct {
	fn         func(u Unmarshaler) error
	eventTypes map[string]reflect.Type
	worker     worker.Worker
}

func (s *subscription) processMessage(ctx context.Context, data string) error {
	var snsMessage awsutil.SNSSQSMessage
	if err := json.Unmarshal([]byte(data), &snsMessage); err != nil {
		return errors.Trace(err)
	}

	resourceName, err := resourceNameFromARN(snsMessage.TopicArn)
	if err != nil {
		return errors.Trace(err)
	}

	eventTypeInstance := newInstanceFromType(s.eventTypes[resourceName])
	if err := json.Unmarshal([]byte(snsMessage.Message), eventTypeInstance); err != nil {
		return errors.Trace(err)
	}

	return errors.Trace(s.fn(eventTypeInstance.(Unmarshaler)))
}
