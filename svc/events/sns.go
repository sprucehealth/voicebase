package events

import (
	"encoding/base64"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

// Publish posts an event to an SNS topic
func Publish(sn snsiface.SNSAPI, topicARN string, svc Service, event Marshaler) {
	envelopeData, err := MarshalEnvelope(svc, event)
	if err != nil {
		golog.Errorf("failed to marshal event envelope %T: %s", event, err)
		return
	}
	golog.Debugf("Publishing event for service %s: %T", svc.String(), event)
	conc.Go(func() {
		if _, err := sn.Publish(&sns.PublishInput{
			Message:  ptr.String(base64.StdEncoding.EncodeToString(envelopeData)),
			TopicArn: ptr.String(topicARN),
		}); err != nil {
			golog.Errorf("failed to publish event: %s", err)
		}
	})
}

// MarshalEnvelope encloses an event in an envelope and marshals it
func MarshalEnvelope(svc Service, event Marshaler) ([]byte, error) {
	eventData, err := event.Marshal()
	if err != nil {
		return nil, errors.Trace(err)
	}
	envelopeData, err := (&Envelope{
		Service: svc,
		Event:   eventData,
	}).Marshal()
	return envelopeData, errors.Trace(err)
}
