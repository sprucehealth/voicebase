package app_url

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

const (
	// action names
	BeginPatientVisitReviewAction   = "begin_patient_visit"
	ViewCompletedPatientVisitAction = "view_completed_patient_visit"
	ViewRefillRequestAction         = "view_refill_request"
	ViewTransmissionErrorAction     = "view_transmission_error"
	ViewPatientTreatmentsAction     = "view_patient_treatments"
	ViewPatientConversationsAction  = "view_patient_conversations"
	ViewPatientVisitAction          = "view_visit"
	ContinueVisitAction             = "continue_visit"
	ViewTreatmentPlanAction         = "view_treatment_plan"
	ViewMessagesAction              = "view_messages"
	ViewCareTeam                    = "view_care_team"

	// asset names
	PatientVisitQueueIcon       = "patient_visit_queue_icon"
	IconBlueTreatmentPlan       = "icon_blue_treatment_plan"
	IconReply                   = "icon_reply"
	IconHomeVisitNormal         = "icon_home_visit_normal"
	IconHomeTreatmentPlanNormal = "icon_home_treatmentplan_normal"
	IconHomeConversationNormal  = "icon_home_conversation_normal"
	IconLogMessage              = "icon_log_message"
)

var registeredSpruceActions = map[string]reflect.Type{}
var registeredSpruceAssets = map[string]reflect.Type{}

func init() {
	registerSpruceAction(BeginPatientVisitReviewAction)
	registerSpruceAction(ViewCompletedPatientVisitAction)
	registerSpruceAction(ViewRefillRequestAction)
	registerSpruceAction(ViewTransmissionErrorAction)
	registerSpruceAction(ViewPatientTreatmentsAction)
	registerSpruceAction(ViewPatientConversationsAction)
	registerSpruceAction(ContinueVisitAction)
	registerSpruceAction(ViewTreatmentPlanAction)
	registerSpruceAction(ViewPatientVisitAction)
	registerSpruceAction(ViewCareTeam)
	registerSpruceAction(ViewMessagesAction)

	registerSpruceAsset(PatientVisitQueueIcon)
	registerSpruceAsset(IconBlueTreatmentPlan)
	registerSpruceAsset(IconReply)
	registerSpruceAsset(IconHomeVisitNormal)
	registerSpruceAsset(IconHomeTreatmentPlanNormal)
	registerSpruceAsset(IconHomeConversationNormal)
	registerSpruceAsset(IconLogMessage)
}

func registerSpruceAction(name string) {
	registeredSpruceActions[name] = reflect.TypeOf(reflect.ValueOf(SpruceAction{}).Interface())
}

func registerSpruceAsset(name string) {
	registeredSpruceAssets[name] = reflect.TypeOf(reflect.ValueOf(SpruceAsset{}).Interface())
}

func GetSpruceActionUrl(actionName string, params url.Values) *SpruceAction {
	s := registeredSpruceActions[actionName]
	sAction := reflect.New(s).Interface().(*SpruceAction)
	sAction.ActionName = actionName
	sAction.params = params
	return sAction
}

func GetSpruceAssetUrl(assetName string) *SpruceAsset {
	s := registeredSpruceAssets[assetName]
	sAsset := reflect.New(s).Interface().(*SpruceAsset)
	sAsset.Name = assetName
	return sAsset
}

func GetLargeThumbnail(role string, id int64) *SpruceAsset {
	return &SpruceAsset{
		Name: fmt.Sprintf("%s_%d_large", strings.ToLower(role), id),
	}
}

func GetSmallThumbnail(role string, id int64) *SpruceAsset {
	return &SpruceAsset{
		Name: fmt.Sprintf("%s_%d_small", strings.ToLower(role), id),
	}
}
