package app_url

import (
	"net/url"
	"reflect"
)

const (
	// action names
	BeginPatientVisitReviewAction   = "begin_patient_visit"
	ViewCompletedPatientVisitAction = "view_completed_patient_visit"
	ViewRefillRequestAction         = "view_refill_request"
	ViewTransmissionErrorAction     = "view_transmission_error"
	ViewPatientTreatmentsAction     = "view_patient_treatments"
	ViewPatientConversationsAction  = "view_patient_conversations"

	// asset names
	PatientVisitQueueIcon = "patient_visit_queue_icon"
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

	registerSpruceAsset(PatientVisitQueueIcon)
}

func registerSpruceAction(name string) {
	registeredSpruceActions[name] = reflect.TypeOf(reflect.ValueOf(spruceAction{}).Interface())
}

func registerSpruceAsset(name string) {
	registeredSpruceAssets[name] = reflect.TypeOf(reflect.ValueOf(spruceAsset{}).Interface())
}

func GetSpruceActionUrl(actionName string, params url.Values) SpruceUrl {
	s := registeredSpruceActions[actionName]
	sAction := reflect.New(s).Interface().(*spruceAction)
	sAction.ActionName = actionName
	sAction.params = params
	return sAction
}

func GetSpruceAssetUrl(assetName string) SpruceUrl {
	s := registeredSpruceAssets[assetName]
	sAsset := reflect.New(s).Interface().(*spruceAsset)
	sAsset.Name = assetName
	return sAsset
}
