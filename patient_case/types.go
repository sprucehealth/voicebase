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

type caseData struct {
	Case            *common.PatientCase
	Notification    *common.CaseNotification
	CareTeamMembers []*common.CareProviderAssignment
	APIDomain       string
}

// notification is an interface for a case notification
// which can be rendered as a notification item within a case file
// as well as a notification home card view on the home tab
type notification interface {
	common.Typed
	canRenderCaseNotificationView() bool
	makeCaseNotificationView(data *caseData) (common.ClientView, error)
	makeHomeCardView(data *caseData) (common.ClientView, error)
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

func (t *treatmentPlanNotification) makeCaseNotificationView(data *caseData) (common.ClientView, error) {

	nView := &caseNotificationMessageView{
		ID:          data.Notification.ID,
		Title:       "Your doctor created your treatment plan.",
		IconURL:     app_url.IconTreatmentPlanSmall,
		ActionURL:   app_url.ViewTreatmentPlanMessageAction(t.MessageID, t.TreatmentPlanID, t.CaseID),
		MessageID:   t.MessageID,
		RoundedIcon: true,
		DateTime:    data.Notification.CreationDate,
	}

	return nView, nView.Validate()
}

func (t *treatmentPlanNotification) makeHomeCardView(data *caseData) (common.ClientView, error) {

	doctorAssignment := findActiveDoctor(data.CareTeamMembers)
	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("%s reviewed your visit and created a treatment plan.", doctorAssignment.ShortDisplayName),
		IconURL:     app_url.LargeThumbnailURL(data.APIDomain, doctorAssignment.ProviderRole, doctorAssignment.ProviderID),
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

func (m *messageNotification) makeCaseNotificationView(data *caseData) (common.ClientView, error) {
	title := "Message from your doctor."
	if m.Role == api.MA_ROLE {
		title = "Message from your care coordinator."
	}

	nView := &caseNotificationMessageView{
		ID:          data.Notification.ID,
		Title:       title,
		IconURL:     app_url.IconMessagesSmall,
		ActionURL:   app_url.ViewCaseMessageAction(m.MessageID, m.CaseID),
		MessageID:   m.MessageID,
		RoundedIcon: true,
		DateTime:    data.Notification.CreationDate,
	}
	return nView, nView.Validate()
}

func (m *messageNotification) makeHomeCardView(data *caseData) (common.ClientView, error) {
	var provider *common.CareProviderAssignment
	for _, assignment := range data.CareTeamMembers {
		if assignment.ProviderID == m.DoctorID {
			provider = assignment
		}
	}

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("You have a new message from %s.", provider.LongDisplayName),
		IconURL:     app_url.LargeThumbnailURL(data.APIDomain, provider.ProviderRole, provider.ProviderID),
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

func (v *visitSubmittedNotification) makeCaseNotificationView(data *caseData) (common.ClientView, error) {
	return nil, nil
}

func (v *visitSubmittedNotification) makeHomeCardView(data *caseData) (common.ClientView, error) {
	title := visitSubmittedSubtitle
	iconURL := app_url.IconVisitSubmitted.String()
	doctorAssignment := findActiveDoctor(data.CareTeamMembers)

	if doctorAssignment != nil {
		title = fmt.Sprintf("%s will review your visit and respond within 24 hours.", doctorAssignment.ShortDisplayName)
		iconURL = app_url.LargeThumbnailURL(data.APIDomain, doctorAssignment.ProviderRole, doctorAssignment.ProviderID)
	}

	nView := &phCaseNotificationStandardView{
		Title:       title,
		IconURL:     iconURL,
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
	continueVisitMessage = "Complete your visit and get a personalized treatment plan from your doctor in 24 hours."
)

func (v *incompleteVisitNotification) canRenderCaseNotificationView() bool { return true }

func (v *incompleteVisitNotification) makeCaseNotificationView(data *caseData) (common.ClientView, error) {

	nView := &caseNotificationTitleSubtitleView{
		Title:     fmt.Sprintf("Continue Your %s Visit", data.Case.Name),
		Subtitle:  continueVisitMessage,
		ID:        data.Notification.ID,
		ActionURL: app_url.ContinueVisitAction(v.PatientVisitID),
	}
	return nView, nView.Validate()
}

func (v *incompleteVisitNotification) makeHomeCardView(data *caseData) (common.ClientView, error) {
	iconURL := app_url.IconCaseLarge.String()
	doctorAssignment := findActiveDoctor(data.CareTeamMembers)

	if doctorAssignment != nil {
		iconURL = app_url.LargeThumbnailURL(data.APIDomain, doctorAssignment.ProviderRole, doctorAssignment.ProviderID)
	}

	nView := &phContinueVisit{
		Title:       fmt.Sprintf("Continue Your %s Visit", data.Case.Name),
		Subtitle:    "You're Almost There!",
		IconURL:     iconURL,
		ActionURL:   app_url.ContinueVisitAction(v.PatientVisitID),
		Description: continueVisitMessage,
		ButtonTitle: "Continue Visit",
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

func (v *incompleteFollowupVisitNotification) makeCaseNotificationView(data *caseData) (common.ClientView, error) {
	nView := &caseNotificationMessageView{
		ID:        data.Notification.ID,
		Title:     "Complete your follow-up visit",
		IconURL:   app_url.IconCaseSmall,
		ActionURL: app_url.ContinueVisitAction(v.PatientVisitID),
		DateTime:  data.Notification.CreationDate,
	}
	return nView, nView.Validate()
}

func (v *incompleteFollowupVisitNotification) makeHomeCardView(data *caseData) (common.ClientView, error) {
	doctorMember := findActiveDoctor(data.CareTeamMembers)

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("Complete your follow-up visit with %s", doctorMember.ShortDisplayName),
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

func (v *startFollowupVisitNotification) makeCaseNotificationView(data *caseData) (common.ClientView, error) {
	nView := &caseNotificationMessageView{
		ID:        data.Notification.ID,
		Title:     "Start your follow-up visit",
		IconURL:   app_url.IconCaseSmall,
		ActionURL: app_url.ContinueVisitAction(v.PatientVisitID),
		DateTime:  data.Notification.CreationDate,
	}
	return nView, nView.Validate()
}

func (v *startFollowupVisitNotification) makeHomeCardView(data *caseData) (common.ClientView, error) {
	doctorMember := findActiveDoctor(data.CareTeamMembers)

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("%s requested a follow-up visit", doctorMember.ShortDisplayName),
		IconURL:     app_url.LargeThumbnailURL(data.APIDomain, doctorMember.ProviderRole, doctorMember.ProviderID),
		ButtonTitle: "View Case",
		ActionURL:   app_url.ViewCaseAction(v.CaseID),
	}

	return nView, nView.Validate()
}

func findActiveDoctor(careTeamMembers []*common.CareProviderAssignment) *common.CareProviderAssignment {
	for _, assignment := range careTeamMembers {
		if assignment.Status == api.STATUS_ACTIVE && assignment.ProviderRole == api.DOCTOR_ROLE {
			return assignment
		}
	}
	return nil
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
