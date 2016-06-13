package setupthread

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	tmock "github.com/sprucehealth/backend/svc/threading/mock"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestWorker(t *testing.T) {
	t.Parallel()

	ts := tmock.New(t)
	w := NewWorker(nil, ts, "")

	ts.Expect(mock.NewExpectation(ts.OnboardingThreadEvent, &threading.OnboardingThreadEventRequest{
		LookupByType: threading.OnboardingThreadEventRequest_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "ent",
		},
		EventType: threading.OnboardingThreadEventRequest_PROVISIONED_PHONE,
		Event: &threading.OnboardingThreadEventRequest_ProvisionedPhone{
			ProvisionedPhone: &threading.ProvisionedPhoneEvent{
				PhoneNumber: "+15551112222",
			},
		},
	}))

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
