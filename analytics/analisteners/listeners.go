package analisteners

import (
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/events/model"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type EventClient interface {
	InsertWebRequestEvent(*model.WebRequestEvent) error
	InsertServerEvent(*model.ServerEvent) error
	InsertClientEvent([]*model.ClientEvent) error
}

func InitListeners(analyticsLogger analytics.Logger, dispatcher *dispatch.Dispatcher, eventsClient EventClient) {
	// Log analytics
	dispatcher.SubscribeAllAsync(func(ev interface{}) error {
		e, ok := ev.(analytics.Eventer)
		if ok {
			eventList := e.Events()
			analyticsLogger.WriteEvents(eventList)
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
		return nil
	})
}
