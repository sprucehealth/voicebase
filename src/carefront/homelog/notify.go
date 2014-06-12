/*
Package homelog provides the implementation of the home feed notifications and log.
*/
package homelog

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_url"
	"carefront/common"
	"fmt"

	"reflect"
)

const (
	bodyButton                   = "body_button"
	incompleteVisit              = "incomplete_visit"
	treatmentPlanCreated         = "treatment_plan_created"
	patientNotificationNamespace = "patient_notification"
	newConversation              = "new_conversation"
	conversationReply            = "conversation_reply"
	message                      = "message"
)

type notification interface {
	common.Typed
	makeView(dataAPI api.DataAPI, patientId, notificationId int64) (view, error)
}

type incompleteVisitNotification struct {
	VisitId int64
}

type treatmentPlanCreatedNotification struct {
	DoctorId        int64
	VisitId         int64
	TreatmentPlanId int64
}

type newConversationNotification struct {
	DoctorId       int64
	ConversationId int64
}

type conversationReplyNotification struct {
	DoctorId       int64
	ConversationId int64
}

func (*incompleteVisitNotification) TypeName() string {
	return incompleteVisit
}

func (*treatmentPlanCreatedNotification) TypeName() string {
	return treatmentPlanCreated
}

func (*newConversationNotification) TypeName() string {
	return newConversation
}

func (*conversationReplyNotification) TypeName() string {
	return conversationReply
}

func (n *incompleteVisitNotification) makeView(dataAPI api.DataAPI, patientId, notificationId int64) (view, error) {
	patient, err := dataAPI.GetPatientFromId(patientId)
	if err != nil {
		return nil, err
	}
	doctor, err := apiservice.GetPrimaryDoctorInfoBasedOnPatient(dataAPI, patient, "")
	if err != nil {
		return nil, err
	}

	return &incompleteVisitView{
		Type:           patientNotificationNamespace + ":" + incompleteVisit,
		Title:          fmt.Sprintf("Complete your visit with Dr. %s.", doctor.LastName),
		IconURL:        doctor.SmallThumbnailUrl,
		ButtonText:     "Continue Your Visit",
		TapURL:         app_url.ContinueVisitAction(n.VisitId),
		PatientVisitId: n.VisitId,
		NotificationId: notificationId,
	}, nil
}

func (n *treatmentPlanCreatedNotification) makeView(dataAPI api.DataAPI, patientId, notificationId int64) (view, error) {
	doctor, err := dataAPI.GetDoctorFromId(n.DoctorId)
	if err != nil {
		return nil, err
	}

	return &bodyButtonView{
		Dismissible:       true,
		DismissOnAction:   true,
		Type:              patientNotificationNamespace + ":" + bodyButton,
		Title:             fmt.Sprintf("Dr. %s created your treatment plan.", doctor.LastName),
		IconURL:           doctor.SmallThumbnailUrl,
		TapURL:            app_url.ViewTreatmentPlanAction(n.TreatmentPlanId),
		BodyButtonIconURL: app_url.IconBlueTreatmentPlan,
		BodyButtonText:    "Treatment Plan",
		BodyButtonTapURL:  app_url.ViewTreatmentPlanAction(n.TreatmentPlanId),
		NotificationId:    notificationId,
	}, nil
}

func (n *newConversationNotification) makeView(dataAPI api.DataAPI, patientId, notificationId int64) (view, error) {
	doctor, err := dataAPI.GetDoctorFromId(n.DoctorId)
	if err != nil {
		return nil, err
	}
	con, err := dataAPI.GetConversation(n.ConversationId)
	if err != nil {
		return nil, err
	}
	return &messageView{
		Dismissible:     true,
		DismissOnAction: true,
		Type:            patientNotificationNamespace + ":" + message,
		Title:           fmt.Sprintf("Dr. %s sent you a message.", doctor.LastName),
		IconURL:         doctor.SmallThumbnailUrl,
		TapURL:          app_url.ViewMessagesAction(n.ConversationId),
		ButtonIconURL:   app_url.IconReply,
		ButtonText:      "Reply",
		Text:            con.Messages[0].Body,
		NotificationId:  notificationId,
	}, nil
}

func (n *conversationReplyNotification) makeView(dataAPI api.DataAPI, patientId, notificationId int64) (view, error) {
	doctor, err := dataAPI.GetDoctorFromId(n.DoctorId)
	if err != nil {
		return nil, err
	}
	con, err := dataAPI.GetConversation(n.ConversationId)
	if err != nil {
		return nil, err
	}

	return &messageView{
		Dismissible:     true,
		DismissOnAction: true,
		Type:            patientNotificationNamespace + ":" + message,
		Title:           fmt.Sprintf("Dr. %s replied to your message about %s.", doctor.LastName, con.Title),
		IconURL:         doctor.SmallThumbnailUrl,
		TapURL:          app_url.ViewMessagesAction(n.ConversationId),
		Text:            con.Messages[len(con.Messages)-1].Body,
		NotificationId:  notificationId,
	}, nil
}

var notifyTypes = map[string]reflect.Type{}

func init() {
	registerNotificationType(&incompleteVisitNotification{})
	registerNotificationType(&treatmentPlanCreatedNotification{})
	registerNotificationType(&newConversationNotification{})
	registerNotificationType(&conversationReplyNotification{})
}

func registerNotificationType(n notification) {
	notifyTypes[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
}
