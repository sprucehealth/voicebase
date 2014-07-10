package patient_case

import (
	"fmt"
	"reflect"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

const (
	CNTreatmentPlan   = "treatment_plan"
	CNMessage         = "message"
	CNVisitSubmitted  = "visit_submitted"
	CNIncompleteVisit = "incomplete_visit"
)

// notification is an interface for a case notification
// which can be rendered as a notification item within a case file
// as well as a notification home card view on the home tab
type notification interface {
	common.Typed
	makeCaseNotificationView(dataAPI api.DataAPI, notification *common.CaseNotification) (common.ClientView, error)
	makeHomeCardView(dataAPI api.DataAPI) (common.ClientView, error)
}

type treatmentPlanNotification struct {
	MessageId       int64 `json:"message_id"`
	DoctorId        int64 `json:"doctor_id"`
	TreatmentPlanId int64 `json:"treatment_plan_id"`
	CaseId          int64 `json:"case_id"`
}

func (t *treatmentPlanNotification) TypeName() string {
	return CNTreatmentPlan
}

func (t *treatmentPlanNotification) makeCaseNotificationView(dataAPI api.DataAPI, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationMessageView{
		ID:          notification.Id,
		Title:       "Your doctor created your treatment plan.",
		IconURL:     app_url.IconTreatmentPlanSmall,
		ActionURL:   app_url.ViewCaseMessageAction(t.MessageId, t.CaseId),
		MessageID:   t.MessageId,
		RoundedIcon: true,
		DateTime:    notification.CreationDate,
	}

	return nView, nView.Validate()
}

func (t *treatmentPlanNotification) makeHomeCardView(dataAPI api.DataAPI) (common.ClientView, error) {
	doctor, err := dataAPI.GetDoctorFromId(t.DoctorId)
	if err != nil {
		return nil, err
	}

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("Dr. %s reviewed your visit and created your treatment plan.", doctor.LastName),
		IconURL:     app_url.IconTreatmentPlanLarge,
		ButtonTitle: "View Treatment Plan",
		ActionURL:   app_url.ViewCaseMessageAction(t.MessageId, t.CaseId),
	}

	return nView, nView.Validate()
}

type messageNotification struct {
	MessageId int64 `json:"message_id"`
	DoctorId  int64 `json:"doctor_id"`
	CaseId    int64 `json:"case_id"`
}

func (m *messageNotification) TypeName() string {
	return CNMessage
}

func (m *messageNotification) makeCaseNotificationView(dataAPI api.DataAPI, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationMessageView{
		ID:          notification.Id,
		Title:       "Message from your doctor.",
		IconURL:     app_url.GetSmallThumbnail(api.DOCTOR_ROLE, m.DoctorId),
		ActionURL:   app_url.ViewCaseMessageAction(m.MessageId, m.CaseId),
		MessageID:   m.MessageId,
		RoundedIcon: true,
		DateTime:    notification.CreationDate,
	}
	return nView, nView.Validate()
}

func (m *messageNotification) makeHomeCardView(dataAPI api.DataAPI) (common.ClientView, error) {
	doctor, err := dataAPI.GetDoctorFromId(m.DoctorId)
	if err != nil {
		return nil, err
	}

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("You have a new message from Dr. %s %s", doctor.FirstName, doctor.LastName),
		IconURL:     app_url.IconMessagesLarge,
		ActionURL:   app_url.ViewCaseMessageAction(m.MessageId, m.CaseId),
		ButtonTitle: "View Message",
	}

	return nView, nView.Validate()
}

type visitSubmittedNotification struct{}

func (v *visitSubmittedNotification) TypeName() string {
	return CNVisitSubmitted
}

const (
	visitSubmittedSubtitle = "Your dermatologist will review your visit and respond within 24 hours."
	visitSubmittedTitle    = "Your acne case has been successfully submitted."
)

func (v *visitSubmittedNotification) makeCaseNotificationView(dataAPI api.DataAPI, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationTitleSubtitleView{
		ID:       notification.Id,
		Title:    visitSubmittedTitle,
		Subtitle: visitSubmittedSubtitle,
	}

	return nView, nView.Validate()
}

func (v *visitSubmittedNotification) makeHomeCardView(dataAPI api.DataAPI) (common.ClientView, error) {
	nView := &phCaseNotificationStandardView{
		Title:    visitSubmittedTitle,
		IconURL:  app_url.IconCheckmarkLarge,
		Subtitle: visitSubmittedSubtitle,
	}

	return nView, nView.Validate()
}

type incompleteVisitNotification struct {
	PatientVisitId int64
}

func (v *incompleteVisitNotification) TypeName() string {
	return CNIncompleteVisit
}

const (
	continueVisitMessage = "You're almost there. Complete your visit and get on the path to clear skin."
	continueVisitTitle   = "Continue Your Acne Visit"
)

func (v *incompleteVisitNotification) makeCaseNotificationView(dataAPI api.DataAPI, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationTitleSubtitleView{
		Title:     continueVisitTitle,
		Subtitle:  continueVisitMessage,
		ID:        notification.Id,
		ActionURL: app_url.ContinueVisitAction(v.PatientVisitId),
	}
	return nView, nView.Validate()
}

func (v *incompleteVisitNotification) makeHomeCardView(dataAPI api.DataAPI) (common.ClientView, error) {
	nView := &phContinueVisit{
		Title:       continueVisitTitle,
		ActionURL:   app_url.ContinueVisitAction(v.PatientVisitId),
		Description: continueVisitMessage,
		ButtonTitle: "Continue",
	}

	return nView, nView.Validate()
}

func init() {
	registerNotificationType(&treatmentPlanNotification{})
	registerNotificationType(&messageNotification{})
	registerNotificationType(&visitSubmittedNotification{})
	registerNotificationType(&incompleteVisitNotification{})
}

var notifyTypes = make(map[string]reflect.Type)

func registerNotificationType(n notification) {
	notifyTypes[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
}
