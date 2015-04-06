package analisteners

import (
	"testing"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/events/model"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/test"
)

type TestEventClient struct {
	InsertWebRequestEventCalled bool
	InsertServerEventCalled     bool
	InsertClientEventCalled     bool
}

func (ec *TestEventClient) InsertWebRequestEvent(*model.WebRequestEvent) error {
	ec.InsertWebRequestEventCalled = true
	return nil
}
func (ec *TestEventClient) InsertServerEvent(*model.ServerEvent) error {
	ec.InsertServerEventCalled = true
	return nil
}
func (ec *TestEventClient) InsertClientEvent([]*model.ClientEvent) error {
	ec.InsertClientEventCalled = true
	return nil
}

type TestLogger struct {
	WriteEventsCalled bool
	StartCalled       bool
	StopCalled        bool
}

func (l *TestLogger) WriteEvents([]analytics.Event) {
	l.WriteEventsCalled = true
	return
}
func (l *TestLogger) Start() error {
	l.StartCalled = true
	return nil
}
func (l *TestLogger) Stop() error {
	l.StopCalled = true
	return nil
}

type eventable struct {
	events []analytics.Event
}

func newEvent(es []analytics.Event) *eventable {
	return &eventable{
		events: es,
	}
}

func (e *eventable) Events() []analytics.Event {
	return e.events
}

func init() {
	dispatch.Testing = true
}

func TestListenersNonWritableEventLogging(t *testing.T) {
	dispatcher := dispatch.New()
	logger := &TestLogger{}
	client := &TestEventClient{}
	InitListeners(logger, dispatcher, client)
	dispatcher.Publish(&struct{}{})
	test.Assert(t, !logger.WriteEventsCalled, "Expected event not matching Eventer interface to not be written but it was.")
	test.Assert(t, !client.InsertClientEventCalled && !client.InsertServerEventCalled && !client.InsertWebRequestEventCalled, "Expected nothing to be inserted into the DB")
}

func TestListenersWritableEventLogging(t *testing.T) {
	dispatcher := dispatch.New()
	logger := &TestLogger{}
	client := &TestEventClient{}
	InitListeners(logger, dispatcher, client)
	dispatcher.Publish(newEvent([]analytics.Event{}))
	test.Assert(t, logger.WriteEventsCalled, "Expected event matching Eventer interface to be written but it was not.")
	test.Assert(t, !client.InsertClientEventCalled && !client.InsertServerEventCalled && !client.InsertWebRequestEventCalled, "Expected nothing to be inserted into the DB")
}

func TestListenersServerEventsInserted(t *testing.T) {
	dispatcher := dispatch.New()
	logger := &TestLogger{}
	client := &TestEventClient{}
	InitListeners(logger, dispatcher, client)
	dispatcher.Publish(newEvent([]analytics.Event{&analytics.ServerEvent{}}))
	test.Assert(t, logger.WriteEventsCalled, "Expected event matching Eventer interface to be written but it was not.")
	test.Assert(t, client.InsertServerEventCalled, "Expected server event to be inserted into the DB but was not.")
	test.Assert(t, !client.InsertClientEventCalled && !client.InsertWebRequestEventCalled, "Expected only server request event to be inserted into the DB")
}

func TestListenersWebRequestEventsInserted(t *testing.T) {
	dispatcher := dispatch.New()
	logger := &TestLogger{}
	client := &TestEventClient{}
	InitListeners(logger, dispatcher, client)
	dispatcher.Publish(newEvent([]analytics.Event{&analytics.WebRequestEvent{}}))
	test.Assert(t, logger.WriteEventsCalled, "Expected event matching Eventer interface to be written but it was not.")
	test.Assert(t, client.InsertWebRequestEventCalled, "Expected web request event to be inserted into the DB but was not.")
	test.Assert(t, !client.InsertClientEventCalled && !client.InsertServerEventCalled, "Expected only web event to be inserted into the DB")
}

func TestListenersClientEventsInserted(t *testing.T) {
	dispatcher := dispatch.New()
	logger := &TestLogger{}
	client := &TestEventClient{}
	InitListeners(logger, dispatcher, client)
	dispatcher.Publish(newEvent([]analytics.Event{&analytics.ClientEvent{}}))
	test.Assert(t, logger.WriteEventsCalled, "Expected event matching Eventer interface to be written but it was not.")
	test.Assert(t, client.InsertClientEventCalled, "Expected client event to be inserted into the DB but was not.")
	test.Assert(t, !client.InsertServerEventCalled && !client.InsertWebRequestEventCalled, "Expected only client event to be inserted into the DB")
}
