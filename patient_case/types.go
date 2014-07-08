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

type treatmentPlanNotification struct {
	MessageId       int64 `json:"message_id"`
	DoctorId        int64 `json:"doctor_id"`
	TreatmentPlanId int64 `json:"treatment_plan_id"`
}

const (
	CNTreatmentPlan  = "treatment_plan"
	CNMessage        = "message"
	CNVisitSubmitted = "visit_submitted"
)

func (t *treatmentPlanNotification) TypeName() string {
	return CNTreatmentPlan
}

func (t *treatmentPlanNotification) makeView(dataAPI api.DataAPI, notificationId int64) (notificationView, error) {
	doctor, err := dataAPI.GetDoctorFromId(t.DoctorId)
	if err != nil {
		return nil, err
	}

	nView := &caseNotificationMessageView{
		ID:          notificationId,
		Title:       fmt.Sprintf("Dr. %s created your treatment plan.", doctor.LastName),
		IconURL:     app_url.IconBlueTreatmentPlan,
		ActionURL:   app_url.ViewCaseMessageAction(t.MessageId),
		MessageID:   t.MessageId,
		RoundedIcon: true,
	}

	return nView, nView.Validate()
}

type messageNotification struct {
	MessageId    int64 `json:"message_id"`
	DoctorId     int64 `json:"doctor_id"`
	DismissOnTap bool  `json:"dismiss_on_tap"`
}

func (m *messageNotification) TypeName() string {
	return CNMessage
}

func (m *messageNotification) makeView(dataAPI api.DataAPI, notificationId int64) (notificationView, error) {
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

type visitSubmittedNotification struct{}

func (v *visitSubmittedNotification) TypeName() string {
	return CNVisitSubmitted
}

func (v *visitSubmittedNotification) makeView(dataAPI api.DataAPI, notificationId int64) (notificationView, error) {
	nView := &caseNotificationTitleSubtitleView{
		ID:       notificationId,
		Title:    "Your acne case has been successfully submitted.",
		Subtitle: "Your dermatologist will review your visit and respond within 24 hours.",
	}

	return nView, nView.Validate()
}

func init() {
	registerNotificationType(&treatmentPlanNotification{})
	registerNotificationType(&messageNotification{})
	registerNotificationType(&visitSubmittedNotification{})
}

var notifyTypes = make(map[string]reflect.Type)

func registerNotificationType(n notification) {
	notifyTypes[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
}
