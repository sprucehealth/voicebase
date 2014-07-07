package patient_case

import (
	"fmt"
	"reflect"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

type notification interface {
	common.Typed
	makeView(dataAPI api.DataAPI, notificationId int64) (notificationView, error)
}

type treatmentPlanNotificationData struct {
	MessageId       int64 `json:"message_id"`
	DoctorId        int64 `json:"doctor_id"`
	TreatmentPlanId int64 `json:"treatment_plan_id"`
}

func (t *treatmentPlanNotificationData) TypeName() string {
	return common.CNTreatmentPlan
}

func (t *treatmentPlanNotificationData) makeView(dataAPI api.DataAPI, notificationId int64) (notificationView, error) {
	nView := &caseNotificationMessageView{
		ID:          notificationId,
		Title:       "Your treatment plan is ready.",
		IconURL:     app_url.IconBlueTreatmentPlan,
		ActionURL:   app_url.ViewCaseMessageAction(t.MessageId),
		MessageID:   t.MessageId,
		RoundedIcon: true,
	}
	return nView, nView.Validate()
}

type messageNotificationData struct {
	MessageId    int64 `json:"message_id"`
	DoctorId     int64 `json:"doctor_id"`
	DismissOnTap bool  `json:"dismiss_on_tap"`
}

func (m *messageNotificationData) TypeName() string {
	return common.CNMessage
}

func (m *messageNotificationData) makeView(dataAPI api.DataAPI, notificationId int64) (notificationView, error) {
	doctor, err := dataAPI.GetDoctorFromId(m.DoctorId)
	if err != nil {
		return nil, err
	}

	nView := &caseNotificationMessageView{
		ID:           notificationId,
		Title:        fmt.Sprintf("Message from Dr. %s", doctor.LastName),
		IconURL:      app_url.GetSmallThumbnail(api.DOCTOR_ROLE, m.DoctorId),
		ActionURL:    app_url.ViewCaseMessageAction(m.MessageId),
		MessageID:    m.MessageId,
		RoundedIcon:  true,
		DismissOnTap: m.DismissOnTap,
	}
	return nView, nView.Validate()
}

func init() {
	registerNotificationType(&treatmentPlanNotificationData{})
	registerNotificationType(&messageNotificationData{})
}

var notifyTypes = make(map[string]reflect.Type)

func registerNotificationType(n notification) {
	notifyTypes[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
}
