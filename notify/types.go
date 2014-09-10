package notify

import (
	"fmt"
	"reflect"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_visit"
)

const (
	notifyVisitSubmittedEmailType       = "notify-visit-submitted"
	notifyTreatmentPlanCreatedEmailType = "notify-treatment-plan-created"
	notifyNewMessageEmailType           = "notify-new-message"
	notifyCaseAssignedEmailType         = "notify-case-assigned"
	notifyVisitRoutedEmailType          = "notify-visit-routed"
	notifyRxTransmissionEmailType       = "notify-rx-transmission"
	notifyRefillRxCreatedEmailType      = "notify-refill-rx-created"
)

func init() {
	email.MustRegisterType(&email.Type{
		Key:  notifyVisitSubmittedEmailType,
		Name: "Visit Submitted Notification",
		TestContext: &visitSubmittedEmailContext{
			Role: api.PATIENT_ROLE,
		},
	})
	email.MustRegisterType(&email.Type{
		Key:  notifyTreatmentPlanCreatedEmailType,
		Name: "Treatment Plan Created Notification",
		TestContext: &treatmentPlanCreatedEmailContext{
			Role: api.PATIENT_ROLE,
		},
	})
	email.MustRegisterType(&email.Type{
		Key:  notifyNewMessageEmailType,
		Name: "New Message Notification",
		TestContext: &newMessageEmailContext{
			Role: api.PATIENT_ROLE,
		},
	})
	email.MustRegisterType(&email.Type{
		Key:  notifyCaseAssignedEmailType,
		Name: "Case Assigned Notification",
		TestContext: &caseAssignedEmailContext{
			Role: api.PATIENT_ROLE,
		},
	})
	email.MustRegisterType(&email.Type{
		Key:  notifyVisitRoutedEmailType,
		Name: "Visit Routed Notification",
		TestContext: &visitRoutedEmailContext{
			Role: api.PATIENT_ROLE,
		},
	})
	email.MustRegisterType(&email.Type{
		Key:  notifyRxTransmissionEmailType,
		Name: "Rx Transmission Notification",
		TestContext: &rxTransmissionEmailContext{
			Role: api.PATIENT_ROLE,
		},
	})
	email.MustRegisterType(&email.Type{
		Key:  notifyRefillRxCreatedEmailType,
		Name: "Refill Rx Created Notification",
		TestContext: &refillRxCreatedEmailContext{
			Role: api.PATIENT_ROLE,
		},
	})
}

type internalNotificationView interface {
	renderEmail(event interface{}) (string, interface{}, error)
}

var eventToInternalNotificationMapping map[reflect.Type]internalNotificationView

func getInternalNotificationViewForEvent(ev interface{}) internalNotificationView {
	return eventToInternalNotificationMapping[reflect.TypeOf(ev)]
}

type patientVisitUnsuitableView struct{}

func (patientVisitUnsuitableView) renderEmail(event interface{}) (string, interface{}, error) {
	visit, ok := event.(*patient_visit.PatientVisitMarkedUnsuitableEvent)
	if !ok {
		return "", nil, fmt.Errorf("Unexpected type: %T", event)
	}
	return unsuitableEmailType, unsuitableEmailTypeContext{PatientVisitID: visit.PatientVisitId}, nil
}

// notificationView interface represents the set of possible ways in which
// a notification can be rendered for communicating with a user.
// The idea is to have a notificationView for each of the events we are about.
type notificationView interface {
	renderEmail(event interface{}, role string) (string, interface{}, error)
	renderSMS(role string) string
	renderPush(role string, notificationConfig *config.NotificationConfig, notificationCount int64) interface{}
}

var eventToNotificationViewMapping map[reflect.Type]notificationView

func getNotificationViewForEvent(ev interface{}) notificationView {
	return eventToNotificationViewMapping[reflect.TypeOf(ev)]
}

func init() {
	eventToNotificationViewMapping = map[reflect.Type]notificationView{
		reflect.TypeOf(&messages.PostEvent{}):                                newMessageNotificationView{},
		reflect.TypeOf(&app_worker.RefillRequestCreatedEvent{}):              refillRxCreatedNotificationView{},
		reflect.TypeOf(&app_worker.RxTransmissionErrorEvent{}):               rxTransmissionErrorNotificationView{},
		reflect.TypeOf(&patient.VisitSubmittedEvent{}):                       visitSubmittedNotificationView{},
		reflect.TypeOf(&patient_visit.PatientVisitMarkedUnsuitableEvent{}):   caseAssignedNotificationView{},
		reflect.TypeOf(&messages.CaseAssignEvent{}):                          caseAssignedNotificationView{},
		reflect.TypeOf(&patient_visit.VisitChargedEvent{}):                   visitRoutedNotificationView{},
		reflect.TypeOf(&doctor_treatment_plan.TreatmentPlanActivatedEvent{}): treatmentPlanCreatedNotificationView{},
	}

	eventToInternalNotificationMapping = map[reflect.Type]internalNotificationView{
		reflect.TypeOf(&patient_visit.PatientVisitMarkedUnsuitableEvent{}): patientVisitUnsuitableView{},
	}
}

type visitSubmittedEmailContext struct {
	Role string
}

type visitSubmittedNotificationView struct{}

func (visitSubmittedNotificationView) renderEmail(event interface{}, role string) (string, interface{}, error) {
	ctx := &visitSubmittedEmailContext{
		Role: role,
	}
	return notifyVisitSubmittedEmailType, ctx, nil
}

func (visitSubmittedNotificationView) renderSMS(role string) string {
	return "You have a new patient visit waiting."
}

func (v visitSubmittedNotificationView) renderPush(role string, notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, v.renderSMS(role), notificationCount)
}

type treatmentPlanCreatedEmailContext struct {
	Role string
}

type treatmentPlanCreatedNotificationView struct{}

func (treatmentPlanCreatedNotificationView) renderEmail(event interface{}, role string) (string, interface{}, error) {
	ctx := &treatmentPlanCreatedEmailContext{
		Role: role,
	}
	return notifyTreatmentPlanCreatedEmailType, ctx, nil
}

func (treatmentPlanCreatedNotificationView) renderSMS(role string) string {
	if role == api.PATIENT_ROLE {
		return "Your doctor has reviewed your case."
	}

	return "A treatment plan was created for a patient."
}

func (v treatmentPlanCreatedNotificationView) renderPush(role string, notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, v.renderSMS(role), notificationCount)
}

type newMessageEmailContext struct {
	Role string
}

type newMessageNotificationView struct{}

func (newMessageNotificationView) renderEmail(event interface{}, role string) (string, interface{}, error) {
	return notifyNewMessageEmailType, &newMessageEmailContext{Role: role}, nil
}

func (newMessageNotificationView) renderSMS(role string) string {
	return "You have a new message."
}

func (n newMessageNotificationView) renderPush(role string, notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, n.renderSMS(role), notificationCount)
}

type caseAssignedEmailContext struct {
	Role string
}

type caseAssignedNotificationView struct{}

func (caseAssignedNotificationView) renderEmail(event interface{}, role string) (string, interface{}, error) {
	return notifyCaseAssignedEmailType, &caseAssignedEmailContext{Role: role}, nil
}

func (caseAssignedNotificationView) renderSMS(role string) string {
	return "A patient case has been assigned to you."
}

func (n caseAssignedNotificationView) renderPush(role string, notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, n.renderSMS(role), notificationCount)
}

type visitRoutedEmailContext struct {
	Role string
}

type visitRoutedNotificationView struct{}

func (visitRoutedNotificationView) renderEmail(event interface{}, role string) (string, interface{}, error) {
	return notifyVisitRoutedEmailType, &visitRoutedEmailContext{Role: role}, nil
}

func (visitRoutedNotificationView) renderSMS(role string) string {
	return "A patient has submitted a visit."
}

func (v visitRoutedNotificationView) renderPush(role string, notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, v.renderSMS(role), notificationCount)
}

type rxTransmissionEmailContext struct {
	Role string
}

type rxTransmissionErrorNotificationView struct{}

func (rxTransmissionErrorNotificationView) renderEmail(event interface{}, role string) (string, interface{}, error) {
	return notifyRxTransmissionEmailType, &rxTransmissionEmailContext{Role: role}, nil
}

func (rxTransmissionErrorNotificationView) renderSMS(role string) string {
	return "There was an error routing prescription to pharmacy"
}

func (r rxTransmissionErrorNotificationView) renderPush(role string, notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, r.renderSMS(role), notificationCount)
}

type refillRxCreatedEmailContext struct {
	Role string
}

type refillRxCreatedNotificationView struct{}

func (refillRxCreatedNotificationView) renderEmail(event interface{}, role string) (string, interface{}, error) {
	return notifyRefillRxCreatedEmailType, &refillRxCreatedEmailContext{Role: role}, nil
}

func (refillRxCreatedNotificationView) renderSMS(role string) string {
	return "You have a new refill request from a patient"
}

func (r refillRxCreatedNotificationView) renderPush(role string, notificationConfig *config.NotificationConfig, notificationCount int64) interface{} {
	return renderNotification(notificationConfig, r.renderSMS(role), notificationCount)
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
