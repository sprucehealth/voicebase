package setupthread

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const lastStep = 3

type SetupThreadClient interface {
	OnboardingThreadEvent(context.Context, *threading.OnboardingThreadEventRequest, ...grpc.CallOption) (*threading.OnboardingThreadEventResponse, error)
}

type Worker struct {
	sqs          sqsiface.SQSAPI
	eventWorker  *awsutil.SQSWorker
	threadingCli SetupThreadClient
}

type snsMessage struct {
	Message []byte
}

func NewWorker(sqs sqsiface.SQSAPI, threadingCli SetupThreadClient, eventQueueURL string) *Worker {
	w := &Worker{
		sqs:          sqs,
		threadingCli: threadingCli,
	}
	w.eventWorker = awsutil.NewSQSWorker(sqs, eventQueueURL, w.processSNSEvent)
	return w
}

func (w *Worker) Start() {
	w.eventWorker.Start()
}

func (w *Worker) Stop(wait time.Duration) {
	w.eventWorker.Stop(wait)
}

func (w *Worker) processSNSEvent(msg string) error {
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
	return w.processEvent(env)
}

func (w *Worker) processEvent(env *events.Envelope) error {
	ctx := context.Background()

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
					LookupByType: threading.OnboardingThreadEventRequest_ENTITY_ID,
					LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
						EntityID: e.ForEntityID,
					},
					EventType: threading.OnboardingThreadEventRequest_PROVISIONED_PHONE,
					Event: &threading.OnboardingThreadEventRequest_ProvisionedPhone{
						ProvisionedPhone: &threading.ProvisionedPhoneEvent{
							PhoneNumber: e.Endpoint,
						},
					},
				})
				if err != nil {
					switch grpc.Code(err) {
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
