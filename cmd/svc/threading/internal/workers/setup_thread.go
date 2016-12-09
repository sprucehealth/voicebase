package workers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var _ worker.Worker = &setupThreadWorker{}

type setupThreadClient interface {
	OnboardingThreadEvent(context.Context, *threading.OnboardingThreadEventRequest, ...grpc.CallOption) (*threading.OnboardingThreadEventResponse, error)
}

type setupThreadWorker struct {
	sqs          sqsiface.SQSAPI
	eventWorker  *awsutil.SQSWorker
	threadingCli setupThreadClient
}

type snsMessage struct {
	Message []byte
}

func newSetupThreadWorker(sqs sqsiface.SQSAPI, threadingCli setupThreadClient, eventQueueURL string) *setupThreadWorker {
	w := &setupThreadWorker{
		sqs:          sqs,
		threadingCli: threadingCli,
	}
	w.eventWorker = awsutil.NewSQSWorker(sqs, eventQueueURL, w.processSNSEvent)
	return w
}

func (w *setupThreadWorker) Start() {
	w.eventWorker.Start()
}

func (w *setupThreadWorker) Started() bool {
	return w.eventWorker.Started()
}

func (w *setupThreadWorker) Stop(wait time.Duration) {
	w.eventWorker.Stop(wait)
}

func (w *setupThreadWorker) processSNSEvent(ctx context.Context, msg string) error {
	var snsMsg snsMessage
	if err := json.Unmarshal([]byte(msg), &snsMsg); err != nil {
		golog.Errorf("Failed to unmarshal sns message: %s", err.Error())
		return nil
	}
	env := &events.Envelope{}
	if err := env.Unmarshal(snsMsg.Message); err != nil {
		golog.Errorf("Failed to unmarshal event envelope: %s", err)
		return nil
	}
	return w.processEvent(ctx, env)
}

func (w *setupThreadWorker) processEvent(ctx context.Context, env *events.Envelope) error {
	switch env.Service {
	case events.Service_EXCOMMS:
		var ev excomms.Event
		if err := ev.Unmarshal(env.Event); err != nil {
			golog.Errorf("Failed to unmarshal excomms event: %s", err)
			return nil
		}
		golog.Debugf("Onboarding: received event from excomms service: %s", ev.Type.String())
		switch ev.Type {
		case excomms.Event_PROVISIONED_ENDPOINT:
			e := ev.GetProvisionedEndpoint()
			switch e.EndpointType {
			case excomms.EndpointType_PHONE:
				_, err := w.threadingCli.OnboardingThreadEvent(ctx, &threading.OnboardingThreadEventRequest{
					LookupByType: threading.ONBOARDING_THREAD_LOOKUP_BY_ENTITY_ID,
					LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
						EntityID: e.ForEntityID,
					},
					EventType: threading.ONBOARDING_THREAD_EVENT_TYPE_PROVISIONED_PHONE,
					Event: &threading.OnboardingThreadEventRequest_ProvisionedPhone{
						ProvisionedPhone: &threading.ProvisionedPhoneEvent{
							PhoneNumber: e.Endpoint,
						},
					},
				})
				if err != nil {
					switch grpc.Code(errors.Cause(err)) {
					case codes.NotFound:
						// Nothing to do, no setup thread
						return nil
					case codes.FailedPrecondition, codes.InvalidArgument:
						// Can't retry these errors
						golog.Errorf("Failed to update onboarding thread when provisioning endpoint for entity %s: %s", e.ForEntityID, err.Error())
						return nil
					}
					return errors.Trace(err)
				}
			}
		}
	default:
		golog.Debugf("Onboarding: received unhandled event from service %s", env.Service.String())
	}

	return nil
}
