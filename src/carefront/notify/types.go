package notify

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/messages"
	"reflect"
)

type notificationView interface {
	renderEmail(event interface{}, dataApi api.DataAPI) string
	renderSMS(event interface{}, dataApi api.DataAPI) string
	renderPush(platform string, event interface{}, dataApi api.DataAPI) string
}

type visitSubmittedNotificationView int64

func (visitSubmittedNotificationView) renderEmail(event interface{}, dataApi api.DataAPI) string {
	// TODO
	return ""
}

func (visitSubmittedNotificationView) renderSMS(event interface{}, dataApi api.DataAPI) string {
	return "SPRUCE: You have a new patient visit waiting."
}

func (visitSubmittedNotificationView) renderPush(platform string, event interface{}, dataApi api.DataAPI) string {
	switch platform {
	case "Android":
	case "iOS":
	}
	return ""
}

type visitReviewedNotificationView int64

func (visitReviewedNotificationView) renderEmail(event interface{}, dataApi api.DataAPI) string {
	// TODO
	return ""
}

func (visitReviewedNotificationView) renderSMS(event interface{}, dataApi api.DataAPI) string {
	return "SPRUCE: There is an update to your case."
}

func (visitReviewedNotificationView) renderPush(platform string, event interface{}, dataApi api.DataAPI) string {
	switch platform {
	case "Android":
	case "iOS":
	}
	return ""
}

type newMessageNotificationView int64

func (newMessageNotificationView) renderEmail(event interface{}, dataApi api.DataAPI) string {
	// TODO
	return ""
}

func (newMessageNotificationView) renderSMS(event interface{}, dataApi api.DataAPI) string {
	return "SPRUCE: You have a new message."
}

func (newMessageNotificationView) renderPush(platform string, event interface{}, dataApi api.DataAPI) string {
	switch platform {
	case "Android":
	case "iOS":
	}
	return ""
}

var eventToNotificationViewMapping map[reflect.Type]notificationView

func init() {
	eventToNotificationViewMapping = map[reflect.Type]notificationView{
		reflect.TypeOf(&apiservice.VisitSubmittedEvent{}):       visitSubmittedNotificationView(0),
		reflect.TypeOf(&apiservice.VisitReviewSubmittedEvent{}): visitReviewedNotificationView(0),
		reflect.TypeOf(&messages.ConversationStartedEvent{}):    newMessageNotificationView(0),
		reflect.TypeOf(&messages.ConversationReplyEvent{}):      newMessageNotificationView(0),
	}
}
