package homelog

import (
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"

	"reflect"
)

const (
	bodyButton                   = "body_button"
	incompleteVisit              = "incomplete_visit"
	treatmentPlanCreated         = "treatment_plan_created"
	patientNotificationNamespace = "patient_notification"
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

func (*incompleteVisitNotification) TypeName() string {
	return incompleteVisit
}

func (*treatmentPlanCreatedNotification) TypeName() string {
	return treatmentPlanCreated
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

var notifyTypes = map[string]reflect.Type{}

func init() {
	registerNotificationType(&incompleteVisitNotification{})
	registerNotificationType(&treatmentPlanCreatedNotification{})
}

func registerNotificationType(n notification) {
	notifyTypes[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
}
