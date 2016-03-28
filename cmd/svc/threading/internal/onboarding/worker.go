package onboarding

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
)

const lastStep = 3

type Worker struct {
	sqs             sqsiface.SQSAPI
	dal             dal.DAL
	webDomain       string
	eventWorker     *awsutil.SQSWorker
	threadingWorker *awsutil.SQSWorker
}

type snsMessage struct {
	Message []byte
}

func NewWorker(sqs sqsiface.SQSAPI, dal dal.DAL, webDomain, eventQueueURL, threadingQueueURL string) *Worker {
	w := &Worker{
		sqs:       sqs,
		dal:       dal,
		webDomain: webDomain,
	}
	w.eventWorker = awsutil.NewSQSWorker(sqs, eventQueueURL, w.processSNSEvent)
	w.threadingWorker = awsutil.NewSQSWorker(sqs, threadingQueueURL, w.processSNSThreadItem)
	return w
}

func (w *Worker) Start() {
	w.eventWorker.Start()
	w.threadingWorker.Start()
}

func (w *Worker) Stop(wait time.Duration) {
	w.eventWorker.Stop(wait)
	w.threadingWorker.Stop(wait)
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

	var entityID string
	var step int
	var skip bool
	var args map[string]string

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
			case excomms.EndpointType_EMAIL:
				entityID = e.ForEntityID
				step = 2
				args = map[string]string{
					"email": e.Endpoint,
				}
			case excomms.EndpointType_PHONE:
				entityID = e.ForEntityID
				step = 1
				args = map[string]string{
					"phoneNumber": e.Endpoint,
				}
			}
		}
	case events.Service_INVITE:
		var ev invite.Event
		if err := ev.Unmarshal(env.Event); err != nil {
			golog.Errorf("Failed to unmarshal invite event: %s", err)
			return nil
		}
		golog.Debugf("Onboarding: received event from invite service: %s", ev.Type.String())
		switch ev.Type {
		case invite.Event_INVITED_COLLEAGUES:
			e := ev.GetInvitedColleagues()
			entityID = e.OrganizationEntityID
			step = 3
		}
	default:
		golog.Debugf("Onboarding: received unhandled event from service %s", env.Service.String())
	}

	if entityID == "" {
		return nil
	}

	return errors.Trace(w.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		state, err := dl.OnboardingStateForEntity(ctx, entityID, dal.ForUpdate)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil
		} else if err != nil {
			return errors.Trace(err)
		}
		if state.Step != step-1 {
			return nil
		}
		thread, err := dl.Thread(ctx, state.ThreadID)
		if err != nil {
			return errors.Trace(err)
		}
		nextMsg, summary, err := Message(step, skip, w.webDomain, thread.OrganizationID, args)
		if err != nil {
			return errors.Trace(err)
		}
		if nextMsg == "" {
			return errors.Trace(fmt.Errorf("Empty next message for onboarding step %d", step))
		}
		_, err = dl.PostMessage(ctx, &dal.PostMessageRequest{
			ThreadID:     state.ThreadID,
			FromEntityID: thread.PrimaryEntityID,
			Internal:     false,
			Text:         nextMsg,
			Summary:      summary,
		})
		if err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(dl.UpdateOnboardingState(ctx, state.ThreadID, &dal.OnboardingStateUpdate{
			Step: &step,
		}))
	}))
}

func (w *Worker) processSNSThreadItem(msg string) error {
	var snsMsg snsMessage
	if err := json.Unmarshal([]byte(msg), &snsMsg); err != nil {
		golog.Errorf("Failed to unmarshal sns message: %s", err)
		return nil
	}
	var pit threading.PublishedThreadItem
	if err := pit.Unmarshal(snsMsg.Message); err != nil {
		golog.Errorf("Failed to unmarshal message: %s", err)
		return nil
	}
	return errors.Trace(w.processThreadItem(&pit))
}

func (w *Worker) processThreadItem(ti *threading.PublishedThreadItem) error {
	golog.Debugf("Onboarding: processing thread item: %+v", ti)
	ctx := context.Background()
	threadID, err := models.ParseThreadID(ti.ThreadID)
	if err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(w.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		state, err := dl.OnboardingState(ctx, threadID, dal.ForUpdate)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil
		} else if err != nil {
			return errors.Trace(err)
		}
		// End of onboarding
		if state.Step >= lastStep {
			return nil
		}
		thread, err := dl.Thread(ctx, state.ThreadID)
		if err != nil {
			return errors.Trace(err)
		}
		step := state.Step + 1
		item := ti.GetItem()
		msg := item.GetMessage()
		text := msg.Text
		text = strings.ToLower(text)
		skip := strings.Contains(text, "skip")
		if !skip {
			return nil
		}
		nextMsg, summary, err := Message(state.Step+1, skip, w.webDomain, thread.OrganizationID, nil)
		if err != nil {
			return errors.Trace(err)
		}
		if nextMsg == "" {
			return errors.Trace(fmt.Errorf("Empty next message for onboarding step %d", step))
		}
		_, err = dl.PostMessage(ctx, &dal.PostMessageRequest{
			ThreadID:     state.ThreadID,
			FromEntityID: thread.PrimaryEntityID,
			Internal:     false,
			Text:         nextMsg,
			Summary:      summary,
		})
		if err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(dl.UpdateOnboardingState(ctx, state.ThreadID, &dal.OnboardingStateUpdate{
			Step: &step,
		}))
	}))
}
