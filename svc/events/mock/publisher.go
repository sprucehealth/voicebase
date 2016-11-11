package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/events"
)

var (
	_ events.Publisher = &mockPublisher{}
)

type mockPublisher struct {
	mock.Expector
}

func NewPublisher(t *testing.T) *mockPublisher {
	return &mockPublisher{
		mock.Expector{
			T: t,
		},
	}
}

func (m *mockPublisher) Publish(ma events.Marshaler) error {
	rets := m.Record(ma)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (m *mockPublisher) PublishAsync(ma events.Marshaler) {
	m.Record(ma)
}
