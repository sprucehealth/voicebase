package notify

import (
	"fmt"
	"reflect"

	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/patient_visit"
)

type internalNotificationView interface {
	renderEmail(event interface{}) (string, string, error)
}

var eventToInternalNotificationMapping map[reflect.Type]internalNotificationView

func getInternalNotificationViewForEvent(ev interface{}) internalNotificationView {
	return eventToInternalNotificationMapping[reflect.TypeOf(ev)]
}

type panicEventView int64

func (panicEventView) renderEmail(event interface{}) (string, string, error) {
	panicEvent, ok := event.(*config.PanicEvent)
	if !ok {
		return "", "", fmt.Errorf("Unexpected type: %T", event)
	}

	subject := fmt.Sprintf("PANIC %s.%s", panicEvent.AppName, panicEvent.Environment)
	body := panicEvent.Body
	return subject, body, nil
}

type patientVisitUnsuitableView int64

func (patientVisitUnsuitableView) renderEmail(event interface{}) (string, string, error) {
	unsuitableVisit, ok := event.(*patient_visit.PatientVisitMarkedUnsuitableEvent)
	if !ok {
		return "", "", fmt.Errorf("Unexpected type: %T", event)
	}

	subject := fmt.Sprintf("Patient Visit %d marked unsuitable for Spruce", unsuitableVisit.PatientVisitId)
	body := "The patient visit id in the subject was marked as unsuitable for Spruce "
	return subject, body, nil
}

// notificationView interface represents the set of possible ways in which
// a notification can be rendered for communicating with a user.
// The idea is to have a notificationView for each of the events we are about.
type notificationView interface {
	renderEmail() string
	renderSMS() string
	renderPush(notificationConfig *config.NotificationConfig, notificationCount int64) interface{}
}

var eventToNotificationViewMapping map[reflect.Type]notificationView

func getNotificationViewForEvent(ev interface{}) notificationView {
	return eventToNotificationViewMapping[reflect.TypeOf(ev)]
}

func init() {
	eventToNotificationViewMapping = map[reflect.Type]notificationView{
		reflect.TypeOf(&patient_visit.VisitSubmittedEvent{}):                 visitSubmittedNotificationView(0),
		reflect.TypeOf(&doctor_treatment_plan.TreatmentPlanActivatedEvent{}): treatmentPlanCreatedNotificationView(0),
		reflect.TypeOf(&messages.PostEvent{}):                                newMessageNotificationView(0),
		reflect.TypeOf(&app_worker.RefillRequestCreatedEvent{}):              refillRxCreatedNotificationView(0),
		reflect.TypeOf(&app_worker.RxTransmissionErrorEvent{}):               rxTransmissionErrorNotificationView(0),
		reflect.TypeOf(&doctor_treatment_plan.TreatmentPlanActivatedEvent{}): treatmentPlanCreatedNotificationView(0),
		reflect.TypeOf(&messages.CaseAssignEvent{}):                          caseAssignedNotificationView(0),
	}

	eventToInternalNotificationMapping = map[reflect.Type]internalNotificationView{
		reflect.TypeOf(&config.PanicEvent{}):                               panicEventView(0),
		reflect.TypeOf(&patient_visit.PatientVisitMarkedUnsuitableEvent{}): patientVisitUnsuitableView(0),
	}
}

type visitSubmittedNotificationView int64

func (visitSubmittedNotificationView) renderEmail() string {
	// TODO
	return ""
}

func (visitSubmittedNotificationView) renderSMS() string {
	return "You have a new patient visit waiting."
}

func (v visitSubmittedNotificationView) renderPush(notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, v.renderSMS(), notificationCount)
}

type treatmentPlanCreatedNotificationView int64

func (treatmentPlanCreatedNotificationView) renderEmail() string {
	// TODO
	return ""
}

func (treatmentPlanCreatedNotificationView) renderSMS() string {
	return "Your doctor has reviewed your case."
}

func (v treatmentPlanCreatedNotificationView) renderPush(notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, v.renderSMS(), notificationCount)
}

type newMessageNotificationView int64

func (newMessageNotificationView) renderEmail() string {
	// TODO
	return ""
}

func (newMessageNotificationView) renderSMS() string {
	return "You have a new message."
}

func (n newMessageNotificationView) renderPush(notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, n.renderSMS(), notificationCount)
}

type caseAssignedNotificationView int64

func (caseAssignedNotificationView) renderEmail() string {
	// TODO
	return ""
}

func (caseAssignedNotificationView) renderSMS() string {
	return "A patient case has been assigned to you."
}

func (n caseAssignedNotificationView) renderPush(notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, n.renderSMS(), notificationCount)
}

type rxTransmissionErrorNotificationView int64

func (rxTransmissionErrorNotificationView) renderEmail() string {
	// TODO
	return ""
}

func (rxTransmissionErrorNotificationView) renderSMS() string {
	return "There was an error routing prescription to pharmacy"
}

func (r rxTransmissionErrorNotificationView) renderPush(notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, r.renderSMS(), notificationCount)
}

type refillRxCreatedNotificationView int64

func (refillRxCreatedNotificationView) renderEmail() string {
	// TODO
	return ""
}

func (refillRxCreatedNotificationView) renderSMS() string {
	return "You have a new refill request from a patient"
}

func (r refillRxCreatedNotificationView) renderPush(notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, r.renderSMS(), notificationCount)
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
