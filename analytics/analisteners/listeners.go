package analisteners

import (
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/events/model"
	"github.com/sprucehealth/backend/libs/dispatch"
)

// EventClient describes the methods that should be exposed by clients wanting to interact with the events system
type EventClient interface {
	InsertWebRequestEvent(*model.WebRequestEvent) error
	InsertServerEvent(*model.ServerEvent) error
	InsertClientEvent([]*model.ClientEvent) error
}

// InitListeners bootstraps the analytics listeners for publishing events from the application
func InitListeners(application string, analyticsLogger analytics.Logger, dispatcher *dispatch.Dispatcher, eventsClient EventClient) {
	// Log analytics
	dispatcher.SubscribeAllAsync(func(ev interface{}) error {
		e, ok := ev.(analytics.Eventer)
		if ok {
			eventList := e.Events()
			if application != "" {
				for i, v := range eventList {
					sev, ok := v.(*analytics.ServerEvent)
					if ok {
						sev.Application = application
						if sev.Event != "" {
							sev.Event = application + "_" + sev.Event
						}
						eventList[i] = sev
					}
				}
			}
			analyticsLogger.WriteEvents(eventList)

			if eventsClient != nil {
				var clientEvents []*model.ClientEvent
				for _, e := range eventList {
					switch et := e.(type) {
					case *analytics.ServerEvent:
						sev := model.TransformServerEvent(et)
						if err := eventsClient.InsertServerEvent(sev); err != nil {
							return err
						}
					case *analytics.WebRequestEvent:
						wev := model.TransformWebRequestEvent(et)
						if err := eventsClient.InsertWebRequestEvent(wev); err != nil {
							return err
						}
					case *analytics.ClientEvent:
						clientEvents = append(clientEvents, model.TransformClientEvent(et))
					}
				}
				if len(clientEvents) > 0 {
					if err := eventsClient.InsertClientEvent(clientEvents); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}
