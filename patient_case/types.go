package patient_case

import (
	"fmt"
	"reflect"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

const (
	CNTreatmentPlan      = "treatment_plan"
	CNMessage            = "message"
	CNVisitSubmitted     = "visit_submitted"
	CNIncompleteVisit    = "incomplete_visit"
	CNIncompleteFollowup = "incomplete_followup"
	CNStartFollowup      = "start_followup"
)

// notification is an interface for a case notification
// which can be rendered as a notification item within a case file
// as well as a notification home card view on the home tab
type notification interface {
	common.Typed
	makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error)
	makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error)
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

func (t *treatmentPlanNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {

	nView := &caseNotificationMessageView{
		ID:          notification.Id,
		Title:       "Your doctor created your treatment plan.",
		IconURL:     app_url.IconTreatmentPlanSmall,
		ActionURL:   app_url.ViewTreatmentPlanMessageAction(t.MessageId, t.TreatmentPlanId, t.CaseId),
		MessageID:   t.MessageId,
		RoundedIcon: true,
		DateTime:    notification.CreationDate,
	}

	return nView, nView.Validate()
}

func (t *treatmentPlanNotification) makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error) {
	doctor, err := dataAPI.GetDoctorFromId(t.DoctorId)
	if err != nil {
		return nil, err
	}

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("%s reviewed your visit and created your treatment plan.", doctor.ShortDisplayName),
		IconURL:     app_url.LargeThumbnailURL(apiDomain, api.DOCTOR_ROLE, t.DoctorId),
		ButtonTitle: "View Case",
		ActionURL:   app_url.ViewCaseAction(t.CaseId),
	}

	return nView, nView.Validate()
}

type messageNotification struct {
	MessageId int64  `json:"message_id"`
	DoctorId  int64  `json:"doctor_id"`
	CaseId    int64  `json:"case_id"`
	Role      string `json:"role"`
}

func (m *messageNotification) TypeName() string {
	return CNMessage
}

func (m *messageNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {
	title := "Message from your doctor."
	if m.Role == api.MA_ROLE {
		title = "Message from your care coordinator."
	}

	nView := &caseNotificationMessageView{
		ID:          notification.Id,
		Title:       title,
		IconURL:     app_url.IconMessagesSmall,
		ActionURL:   app_url.ViewCaseMessageAction(m.MessageId, m.CaseId),
		MessageID:   m.MessageId,
		RoundedIcon: true,
		DateTime:    notification.CreationDate,
	}
	return nView, nView.Validate()
}

func (m *messageNotification) makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error) {
	doctor, err := dataAPI.GetDoctorFromId(m.DoctorId)
	if err != nil {
		return nil, err
	}

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("You have a new message from %s", doctor.LongDisplayName),
		IconURL:     app_url.LargeThumbnailURL(apiDomain, api.DOCTOR_ROLE, doctor.DoctorId.Int64()),
		ActionURL:   app_url.ViewCaseAction(m.CaseId),
		ButtonTitle: "View Case",
	}

	return nView, nView.Validate()
}

type visitSubmittedNotification struct {
	CaseID int64 `json:"case_id"`
}

func (v *visitSubmittedNotification) TypeName() string {
	return CNVisitSubmitted
}

const (
	visitSubmittedSubtitle = "Your dermatologist will review your visit and respond within 24 hours."
	visitSubmittedTitle    = "Your acne case has been successfully submitted."
)

func (v *visitSubmittedNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationTitleSubtitleView{
		ID:       notification.Id,
		Title:    visitSubmittedTitle,
		Subtitle: visitSubmittedSubtitle,
	}

	return nView, nView.Validate()
}

func (v *visitSubmittedNotification) makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error) {
	title := visitSubmittedSubtitle

	doctorMember, err := dataAPI.GetActiveCareTeamMemberForCase(api.DOCTOR_ROLE, v.CaseID)
	if err != api.NoRowsError && err != nil {
		return nil, err
	} else if err == nil && doctorMember != nil {
		doctor, err := dataAPI.GetDoctorFromId(doctorMember.ProviderID)
		if err != nil {
			return nil, err
		}
		title = fmt.Sprintf("%s will review your visit and respond within 24 hours.", doctor.ShortDisplayName)
	}

	nView := &phCaseNotificationStandardView{
		Title:       title,
		IconURL:     app_url.IconVisitSubmitted.String(),
		ButtonTitle: "View Case",
		ActionURL:   app_url.ViewCaseAction(v.CaseID),
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

func (v *incompleteVisitNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationTitleSubtitleView{
		Title:     continueVisitTitle,
		Subtitle:  continueVisitMessage,
		ID:        notification.Id,
		ActionURL: app_url.ContinueVisitAction(v.PatientVisitId),
	}
	return nView, nView.Validate()
}

func (v *incompleteVisitNotification) makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error) {
	nView := &phContinueVisit{
		Title:       continueVisitTitle,
		ActionURL:   app_url.ContinueVisitAction(v.PatientVisitId),
		Description: continueVisitMessage,
		ButtonTitle: "Continue",
	}

	return nView, nView.Validate()
}

type incompleteFollowupVisitNotification struct {
	PatientVisitID int64
	CaseID         int64
}

func (v *incompleteFollowupVisitNotification) TypeName() string {
	return CNIncompleteFollowup
}

func (v *incompleteFollowupVisitNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationMessageView{
		ID:        notification.Id,
		Title:     "Complete your follow-up visit",
		IconURL:   app_url.IconCaseSmall,
		ActionURL: app_url.ContinueVisitAction(v.PatientVisitID),
		DateTime:  notification.CreationDate,
	}
	return nView, nView.Validate()
}

func (v *incompleteFollowupVisitNotification) makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error) {
	doctorMember, err := dataAPI.GetActiveCareTeamMemberForCase(api.DOCTOR_ROLE, v.CaseID)
	if err != nil {
		return nil, err
	}

	doctor, err := dataAPI.GetDoctorFromId(doctorMember.ProviderID)
	if err != nil {
		return nil, err
	}

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("Complete your follow-up visit with %s", doctor.ShortDisplayName),
		IconURL:     app_url.IconCaseLarge.String(),
		ButtonTitle: "View Case",
		ActionURL:   app_url.ViewCaseAction(v.CaseID),
	}

	return nView, nView.Validate()
}

func init() {
	registerNotificationType(&treatmentPlanNotification{})
	registerNotificationType(&messageNotification{})
	registerNotificationType(&visitSubmittedNotification{})
	registerNotificationType(&incompleteVisitNotification{})
	registerNotificationType(&incompleteFollowupVisitNotification{})
	registerNotificationType(&startFollowupVisitNotification{})
}

type startFollowupVisitNotification struct {
	PatientVisitID int64
	CaseID         int64
}

func (v *startFollowupVisitNotification) TypeName() string {
	return CNStartFollowup
}

func (v *startFollowupVisitNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationMessageView{
		ID:        notification.Id,
		Title:     "Start your follow-up visit",
		IconURL:   app_url.IconCaseSmall,
		ActionURL: app_url.ContinueVisitAction(v.PatientVisitID),
		DateTime:  notification.CreationDate,
	}
	return nView, nView.Validate()
}

func (v *startFollowupVisitNotification) makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error) {
	doctorMember, err := dataAPI.GetActiveCareTeamMemberForCase(api.DOCTOR_ROLE, v.CaseID)
	if err != nil {
		return nil, err
	}

	doctor, err := dataAPI.GetDoctorFromId(doctorMember.ProviderID)
	if err != nil {
		return nil, err
	}

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("%s requested a follow-up visit", doctor.ShortDisplayName),
		IconURL:     app_url.IconCaseLarge.String(),
		ButtonTitle: "View Case",
		ActionURL:   app_url.ViewCaseAction(v.CaseID),
	}

	return nView, nView.Validate()
}

var NotifyTypes = make(map[string]reflect.Type)

func registerNotificationType(n notification) {
	NotifyTypes[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
}
