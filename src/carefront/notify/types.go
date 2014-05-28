package notify

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_worker"
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
	IOSSandBox     *iOSPushNotification     `json:"APNS_SANDBOX,omitempty"`
	IOS            *iOSPushNotification     `json:"APNS,omitempty"`
	Android        *androidPushNotification `json:"GCM,omitempty"`
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
	return "You have a new patient visit waiting."
}

func (v visitSubmittedNotificationView) renderPush(notificationConfig *config.NotificationConfig, event interface{}, dataApi api.DataAPI, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, v.renderSMS(event, dataApi), notificationCount)
}

type visitReviewedNotificationView int64

func (visitReviewedNotificationView) renderEmail(event interface{}, dataApi api.DataAPI) string {
	// TODO
	return ""
}

func (visitReviewedNotificationView) renderSMS(event interface{}, dataApi api.DataAPI) string {
	return "Doctor has reviewed your case."
}

func (v visitReviewedNotificationView) renderPush(notificationConfig *config.NotificationConfig, event interface{}, dataApi api.DataAPI, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, v.renderSMS(event, dataApi), notificationCount)
}

type newMessageNotificationView int64

func (newMessageNotificationView) renderEmail(event interface{}, dataApi api.DataAPI) string {
	// TODO
	return ""
}

func (newMessageNotificationView) renderSMS(event interface{}, dataApi api.DataAPI) string {
	return "You have a new message."
}

func (n newMessageNotificationView) renderPush(notificationConfig *config.NotificationConfig, event interface{}, dataApi api.DataAPI, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, n.renderSMS(event, dataApi), notificationCount)
}

type rxTransmissionErrorNotificationView int64

func (rxTransmissionErrorNotificationView) renderEmail(event interface{}, dataApi api.DataAPI) string {
	// TODO
	return ""
}

func (rxTransmissionErrorNotificationView) renderSMS(event interface{}, dataApi api.DataAPI) string {
	return "There was an error routing prescription to pharmacy"
}

func (r rxTransmissionErrorNotificationView) renderPush(notificationConfig *config.NotificationConfig, event interface{}, dataApi api.DataAPI, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, r.renderSMS(event, dataApi), notificationCount)
}

type refillRxCreatedNotificationView int64

func (refillRxCreatedNotificationView) renderEmail(event interface{}, dataApi api.DataAPI) string {
	// TODO
	return ""
}

func (refillRxCreatedNotificationView) renderSMS(event interface{}, dataApi api.DataAPI) string {
	return "You have a new refill request from a patient"
}

func (r refillRxCreatedNotificationView) renderPush(notificationConfig *config.NotificationConfig, event interface{}, dataApi api.DataAPI, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, r.renderSMS(event, dataApi), notificationCount)
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
		reflect.TypeOf(&app_worker.RefillRequestCreatedEvent{}): refillRxCreatedNotificationView(0),
		reflect.TypeOf(&app_worker.RxTransmissionErrorEvent{}):  rxTransmissionErrorNotificationView(0),
	}
}

func renderNotification(notificationConfig *config.NotificationConfig, message string, badgeCount int64) *snsNotification {
	snsNote := &snsNotification{
		DefaultMessage: message,
	}
	switch notificationConfig.Platform {
	case common.Android:
		snsNote.Android = &androidPushNotification{
			Message: snsNote.DefaultMessage,
		}

	case common.IOS:
		iosNotification := &iOSPushNotification{
			Badge: badgeCount,
			Alert: snsNote.DefaultMessage,
		}
		if notificationConfig.IsApnsSandbox {
			snsNote.IOSSandBox = iosNotification
		} else {
			snsNote.IOS = iosNotification
		}
	}

	return snsNote
}
