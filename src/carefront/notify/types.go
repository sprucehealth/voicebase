package notify

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/common/config"
	"carefront/messages"
	"reflect"
)

type notificationView interface {
	renderEmail(event interface{}, dataApi api.DataAPI) string
	renderSMS(event interface{}, dataApi api.DataAPI) string
	renderPush(notificationConfig *config.NotificationConfig, event interface{}, dataApi api.DataAPI, notificationCount int64) interface{}
}

type snsNotification struct {
	DefaultMessage string                   `json:"default"`
	iosSandBox     *iOSPushNotification     `json:"APNS_SANDBOX,omitempty"`
	ios            *iOSPushNotification     `json:"APNS,omitempty"`
	android        *androidPushNotification `json:"GCM,omitempty"`
}

type iOSPushNotification struct {
	Alert string `json:"alert"`
	Badge int64  `json:"badge,omitempty"`
}

type androidPushNotification struct {
	Message string `json:"message"`
	Url     string `json:"url"`
}

type visitSubmittedNotificationView int64

func (visitSubmittedNotificationView) renderEmail(event interface{}, dataApi api.DataAPI) string {
	// TODO
	return ""
}

func (visitSubmittedNotificationView) renderSMS(event interface{}, dataApi api.DataAPI) string {
	return "SPRUCE: You have a new patient visit waiting."
}

func (v visitSubmittedNotificationView) renderPush(notificationConfig *config.NotificationConfig, event interface{}, dataApi api.DataAPI, notificationCount int64) interface{} {
	snsNote := &snsNotification{
		DefaultMessage: v.renderSMS(event, dataApi),
	}
	switch notificationConfig.Platform {
	case common.Android:
		snsNote.android = &androidPushNotification{
			Message: snsNote.DefaultMessage,
		}

	case common.IOS:
		iosNotification := &iOSPushNotification{
			Badge: notificationCount,
			Alert: snsNote.DefaultMessage,
		}
		if notificationConfig.IsApnsSandbox {
			snsNote.iosSandBox = iosNotification
		} else {
			snsNote.ios = iosNotification
		}
	}

	return snsNote
}

type visitReviewedNotificationView int64

func (visitReviewedNotificationView) renderEmail(event interface{}, dataApi api.DataAPI) string {
	// TODO
	return ""
}

func (visitReviewedNotificationView) renderSMS(event interface{}, dataApi api.DataAPI) string {
	return "SPRUCE: There is an update to your case."
}

func (v visitReviewedNotificationView) renderPush(notificationConfig *config.NotificationConfig, event interface{}, dataApi api.DataAPI, notificationCount int64) interface{} {
	snsNote := &snsNotification{
		DefaultMessage: v.renderSMS(event, dataApi),
	}
	switch notificationConfig.Platform {
	case common.Android:
		snsNote.android = &androidPushNotification{
			Message: snsNote.DefaultMessage,
		}

	case common.IOS:
		iosNotification := &iOSPushNotification{
			Badge: notificationCount,
			Alert: snsNote.DefaultMessage,
		}
		if notificationConfig.IsApnsSandbox {
			snsNote.iosSandBox = iosNotification
		} else {
			snsNote.ios = iosNotification
		}
	}

	return snsNote
}

type newMessageNotificationView int64

func (newMessageNotificationView) renderEmail(event interface{}, dataApi api.DataAPI) string {
	// TODO
	return ""
}

func (newMessageNotificationView) renderSMS(event interface{}, dataApi api.DataAPI) string {
	return "SPRUCE: You have a new message."
}

func (n newMessageNotificationView) renderPush(notificationConfig *config.NotificationConfig, event interface{}, dataApi api.DataAPI, notificationCount int64) interface{} {
	snsNote := &snsNotification{
		DefaultMessage: n.renderSMS(event, dataApi),
	}
	switch notificationConfig.Platform {
	case common.Android:
		snsNote.android = &androidPushNotification{
			Message: snsNote.DefaultMessage,
		}

	case common.IOS:
		iosNotification := &iOSPushNotification{
			Badge: notificationCount,
			Alert: snsNote.DefaultMessage,
		}
		if notificationConfig.IsApnsSandbox {
			snsNote.iosSandBox = iosNotification
		} else {
			snsNote.ios = iosNotification
		}
	}

	return snsNote
}

var eventToNotificationViewMapping map[reflect.Type]notificationView

func getNotificationViewForEvent(ev interface{}) notificationView {
	return eventToNotificationViewMapping[reflect.TypeOf(ev)]
}

func init() {
	eventToNotificationViewMapping = map[reflect.Type]notificationView{
		reflect.TypeOf(&apiservice.VisitSubmittedEvent{}):       visitSubmittedNotificationView(0),
		reflect.TypeOf(&apiservice.VisitReviewSubmittedEvent{}): visitReviewedNotificationView(0),
		reflect.TypeOf(&messages.ConversationStartedEvent{}):    newMessageNotificationView(0),
		reflect.TypeOf(&messages.ConversationReplyEvent{}):      newMessageNotificationView(0),
	}
}
