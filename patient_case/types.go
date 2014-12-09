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
	canRenderCaseNotificationView() bool
	makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error)
	makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error)
}

type treatmentPlanNotification struct {
	MessageID       int64 `json:"message_id"`
	DoctorID        int64 `json:"doctor_id"`
	TreatmentPlanID int64 `json:"treatment_plan_id"`
	CaseID          int64 `json:"case_id"`
}

func (t *treatmentPlanNotification) TypeName() string {
	return CNTreatmentPlan
}

func (t *treatmentPlanNotification) canRenderCaseNotificationView() bool { return true }

func (t *treatmentPlanNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {

	nView := &caseNotificationMessageView{
		ID:          notification.ID,
		Title:       "Your doctor created your treatment plan.",
		IconURL:     app_url.IconTreatmentPlanSmall,
		ActionURL:   app_url.ViewTreatmentPlanMessageAction(t.MessageID, t.TreatmentPlanID, t.CaseID),
		MessageID:   t.MessageID,
		RoundedIcon: true,
		DateTime:    notification.CreationDate,
	}

	return nView, nView.Validate()
}

func (t *treatmentPlanNotification) makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error) {
	doctor, err := dataAPI.GetDoctorFromID(t.DoctorID)
	if err != nil {
		return nil, err
	}

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("%s reviewed your visit and created your treatment plan.", doctor.ShortDisplayName),
		IconURL:     app_url.LargeThumbnailURL(apiDomain, api.DOCTOR_ROLE, t.DoctorID),
		ButtonTitle: "View Case",
		ActionURL:   app_url.ViewCaseAction(t.CaseID),
	}

	return nView, nView.Validate()
}

type messageNotification struct {
	MessageID int64  `json:"message_id"`
	DoctorID  int64  `json:"doctor_id"`
	CaseID    int64  `json:"case_id"`
	Role      string `json:"role"`
}

func (m *messageNotification) TypeName() string {
	return CNMessage
}

func (m *messageNotification) canRenderCaseNotificationView() bool { return true }

func (m *messageNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {
	title := "Message from your doctor."
	if m.Role == api.MA_ROLE {
		title = "Message from your care coordinator."
	}

	nView := &caseNotificationMessageView{
		ID:          notification.ID,
		Title:       title,
		IconURL:     app_url.IconMessagesSmall,
		ActionURL:   app_url.ViewCaseMessageAction(m.MessageID, m.CaseID),
		MessageID:   m.MessageID,
		RoundedIcon: true,
		DateTime:    notification.CreationDate,
	}
	return nView, nView.Validate()
}

func (m *messageNotification) makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error) {
	doctor, err := dataAPI.GetDoctorFromID(m.DoctorID)
	if err != nil {
		return nil, err
	}

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("You have a new message from %s", doctor.LongDisplayName),
		IconURL:     app_url.LargeThumbnailURL(apiDomain, api.DOCTOR_ROLE, doctor.DoctorID.Int64()),
		ActionURL:   app_url.ViewCaseAction(m.CaseID),
		ButtonTitle: "View Case",
	}

	return nView, nView.Validate()
}

type visitSubmittedNotification struct {
	CaseID  int64 `json:"case_id"`
	VisitID int64 `json:"visit_id"`
}

func (v *visitSubmittedNotification) TypeName() string {
	return CNVisitSubmitted
}

const (
	visitSubmittedSubtitle = "Your dermatologist will review your visit and respond within 24 hours."
	visitSubmittedTitle    = "Your acne case has been successfully submitted."
)

func (v *visitSubmittedNotification) canRenderCaseNotificationView() bool { return false }

func (v *visitSubmittedNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {
	return nil, nil
}

func (v *visitSubmittedNotification) makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error) {
	title := visitSubmittedSubtitle

	doctorMember, err := dataAPI.GetActiveCareTeamMemberForCase(api.DOCTOR_ROLE, v.CaseID)
	if err != api.NoRowsError && err != nil {
		return nil, err
	} else if err == nil && doctorMember != nil {
		doctor, err := dataAPI.GetDoctorFromID(doctorMember.ProviderID)
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
	PatientVisitID int64 `json:"PatientVisitId"`
}

func (v *incompleteVisitNotification) TypeName() string {
	return CNIncompleteVisit
}

const (
	continueVisitMessage = "You're almost there. Complete your visit and get on the path to clear skin."
	continueVisitTitle   = "Continue Your Acne Visit"
)

func (v *incompleteVisitNotification) canRenderCaseNotificationView() bool { return true }

func (v *incompleteVisitNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationTitleSubtitleView{
		Title:     continueVisitTitle,
		Subtitle:  continueVisitMessage,
		ID:        notification.ID,
		ActionURL: app_url.ContinueVisitAction(v.PatientVisitID),
	}
	return nView, nView.Validate()
}

func (v *incompleteVisitNotification) makeHomeCardView(dataAPI api.DataAPI, apiDomain string) (common.ClientView, error) {
	nView := &phContinueVisit{
		Title:       continueVisitTitle,
		ActionURL:   app_url.ContinueVisitAction(v.PatientVisitID),
		Description: continueVisitMessage,
		ButtonTitle: "Continue",
	}

	return nView, nView.Validate()
}

type incompleteFollowupVisitNotification struct {
	PatientVisitID int64 `json:"PatientVisitID"`
	CaseID         int64 `json:"CaseID"`
}

func (v *incompleteFollowupVisitNotification) TypeName() string {
	return CNIncompleteFollowup
}

func (v *incompleteFollowupVisitNotification) canRenderCaseNotificationView() bool { return true }

func (v *incompleteFollowupVisitNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationMessageView{
		ID:        notification.ID,
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

	doctor, err := dataAPI.GetDoctorFromID(doctorMember.ProviderID)
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

type startFollowupVisitNotification struct {
	PatientVisitID int64 `json:"PatientVisitID"`
	CaseID         int64 `json:"CaseID"`
}

func (v *startFollowupVisitNotification) TypeName() string {
	return CNStartFollowup
}

func (v *startFollowupVisitNotification) canRenderCaseNotificationView() bool { return true }

func (v *startFollowupVisitNotification) makeCaseNotificationView(dataAPI api.DataAPI, apiDomain string, notification *common.CaseNotification) (common.ClientView, error) {
	nView := &caseNotificationMessageView{
		ID:        notification.ID,
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

	doctor, err := dataAPI.GetDoctorFromID(doctorMember.ProviderID)
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

func init() {
	registerNotificationType(&treatmentPlanNotification{})
	registerNotificationType(&messageNotification{})
	registerNotificationType(&visitSubmittedNotification{})
	registerNotificationType(&incompleteVisitNotification{})
	registerNotificationType(&incompleteFollowupVisitNotification{})
	registerNotificationType(&startFollowupVisitNotification{})
}

var NotifyTypes = make(map[string]reflect.Type)

func registerNotificationType(n notification) {
	NotifyTypes[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
}
