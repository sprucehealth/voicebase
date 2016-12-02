package workers

import (
	"testing"

	"context"

	"github.com/golang/mock/gomock"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/svc/threading/threadingmock"
)

func TestWorker(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	ts := threadingmock.NewMockThreadsClient(ctrl)
	defer ctrl.Finish()
	w := newSetupThreadWorker(nil, ts, "")

	gomock.InOrder(
		ts.EXPECT().OnboardingThreadEvent(context.Background(), &threading.OnboardingThreadEventRequest{
			LookupByType: threading.ONBOARDING_THREAD_LOOKUP_BY_ENTITY_ID,
			LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
				EntityID: "ent",
			},
			EventType: threading.ONBOARDING_THREAD_EVENT_TYPE_PROVISIONED_PHONE,
			Event: &threading.OnboardingThreadEventRequest_ProvisionedPhone{
				ProvisionedPhone: &threading.ProvisionedPhoneEvent{
					PhoneNumber: "+15551112222",
				},
			},
		}),
	)

	test.OK(t, w.processEvent(context.Background(), &events.Envelope{
		Service: events.Service_EXCOMMS,
		Event: serializeEvent(t, &excomms.Event{
			Type: excomms.Event_PROVISIONED_ENDPOINT,
			Details: &excomms.Event_ProvisionedEndpoint{
				ProvisionedEndpoint: &excomms.ProvisionedEndpoint{
					ForEntityID:  "ent",
					EndpointType: excomms.EndpointType_PHONE,
					Endpoint:     "+15551112222",
				},
			},
		}),
	}))
}

type marshaler interface {
	Marshal() ([]byte, error)
}

func serializeEvent(t *testing.T, m marshaler) []byte {
	data, err := m.Marshal()
	test.OK(t, err)
	return data
}
