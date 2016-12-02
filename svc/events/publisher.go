package events

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

// Marshaler is implemented by protocol buffer structs
type Marshaler interface {
	Marshal() ([]byte, error)
}

// Publisher is implemented by any object that supports publishing events
// in the system
type Publisher interface {
	Publish(m Marshaler) error
	PublishAsync(m Marshaler)
}

type snsPublisher struct {
	snsAPI         snsiface.SNSAPI
	topicARNPrefix string
}

// NewSNSPublisher returns a publisher that can be used to publish system events to sns topics
func NewSNSPublisher(snsAPI snsiface.SNSAPI, awsSession *session.Session) (Publisher, error) {
	accountID, err := getAWSAccountID(awsSession)
	if err != nil {
		return nil, errors.Trace(err)
	}

	topicARNPrefix := fmt.Sprintf("arn:aws:sns:%s:%s:", *awsSession.Config.Region, accountID)

	return &snsPublisher{
		snsAPI:         snsAPI,
		topicARNPrefix: topicARNPrefix,
	}, nil
}

// Publish syncrhonously publishes the event to an SNS topic with a name in the following format:
// {env}-{svc}-{eventName}. The assumption here is that the package where the event is defined is the name of the
// service.
func (s *snsPublisher) Publish(m Marshaler) error {
	return s.publish(m)
}

// Publish asyncrhonously publishes the event to an SNS topic with a name in the following format:
// {env}-{svc}-{eventName}. The assumption here is that the package where the event is defined is the name of the
// service.
func (s *snsPublisher) PublishAsync(m Marshaler) {
	conc.Go(func() {
		if err := s.publish(m); err != nil {
			golog.Errorf(err.Error())
		}
	})
}

func (s *snsPublisher) publish(m Marshaler) error {
	eventName := strings.ToLower(nameOfEvent(m))
	topicARN := s.topicARNPrefix + resourceNameFromEvent(m)

	golog.Debugf("Publishing event %s to topic %s", eventName, topicARN)

	eventData, err := m.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	publishFn := func() error {
		_, err := s.snsAPI.Publish(&sns.PublishInput{
			Message:  ptr.String(base64.StdEncoding.EncodeToString(eventData)),
			TopicArn: ptr.String(topicARN),
		})
		return err
	}
	err = publishFn()
	// If we can't find the topic create it. Don't use the CreateSNSTopicIfNotExists helper to avoid an extra call
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == awsutil.AWSErrCodeSNSTopicNotFound {
			topicName, err := awsutil.ResourceNameFromARN(topicARN)
			if err != nil {
				return errors.Errorf("failed to publish event %s to topic since it was NOT FOUND. Unable to get topic name from ARN %s to create due to: %s", eventName, topicARN, err)
			}
			golog.Infof("Topic %s was NOT FOUND. Attempting to create it.", topicARN)
			topicARN, err = awsutil.CreateSNSTopic(s.snsAPI, topicName)
			if err != nil {
				return errors.Errorf("failed to publish event %s to topic since it was NOT FOUND. Failed to create topic due to: %s", eventName, err)
			}
			golog.Infof("Topic %s was successfully created", topicName)
			if err := publishFn(); err != nil {
				return errors.Errorf("failed to publish event %s to topic %s: %s", eventName, topicARN, err)
			}
		} else {
			return errors.Errorf("failed to publish event %s to topic %s: %s", eventName, topicARN, err)
		}
	} else if err != nil {
		return errors.Errorf("failed to publish event %s to topic %s: %s", eventName, topicARN, err)
	}

	return nil
}
