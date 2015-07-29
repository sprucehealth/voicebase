package patient_case

import (
	"fmt"
	"reflect"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/patient"
)

// Prefixes for generating unique IDs for push notifications.
const (
	CNIncompleteFollowup       = "incomplete_followup"
	CNIncompleteVisit          = "incomplete_visit"
	CNMessage                  = "message"
	CNParentalConsentCompleted = "parental_consent_completed"
	CNPreSubmissionTriage      = "pre_submission_triage"
	CNStartFollowup            = "start_followup"
	CNTreatmentPlan            = "treatment_plan"
	CNVisitSubmitted           = "visit_submitted"
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
	makeHomeCardView(dataAPI api.DataAPI, webDomain string, data *caseData) (common.ClientView, error)
}

//
// ****************** treatment plan notification ******************
//

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
		ID:            data.Notification.ID,
		Title:         "Your doctor created your treatment plan.",
		IconURL:       app_url.IconTreatmentPlanSmall,
		ActionURL:     app_url.ViewTreatmentPlanMessageAction(t.MessageID, t.TreatmentPlanID, t.CaseID),
		MessageID:     t.MessageID,
		RoundedIcon:   true,
		CreationTime:  data.Notification.CreationDate,
		CreationEpoch: data.Notification.CreationDate.Unix(),
	}

	return nView, nView.Validate()
}

func (t *treatmentPlanNotification) makeHomeCardView(dataAPI api.DataAPI, webDomain string, data *caseData) (common.ClientView, error) {

	doctorAssignment := findActiveDoctor(data.CareTeamMembers)
	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("%s reviewed your visit and created a treatment plan.", doctorAssignment.ShortDisplayName),
		IconURL:     app_url.ThumbnailURL(data.APIDomain, doctorAssignment.ProviderRole, doctorAssignment.ProviderID),
		ButtonTitle: "View Case",
		ActionURL:   app_url.ViewCaseAction(t.CaseID),
	}

	return nView, nView.Validate()
}

//
// *************** message notification ****************
//

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
	if m.Role == api.RoleCC {
		title = "Message from your care coordinator."
	}

	nView := &caseNotificationMessageView{
		ID:            data.Notification.ID,
		Title:         title,
		IconURL:       app_url.IconMessagesSmall,
		ActionURL:     app_url.ViewCaseMessageAction(m.MessageID, m.CaseID),
		MessageID:     m.MessageID,
		RoundedIcon:   true,
		CreationTime:  data.Notification.CreationDate,
		CreationEpoch: data.Notification.CreationDate.Unix(),
	}
	return nView, nView.Validate()
}

func (m *messageNotification) makeHomeCardView(dataAPI api.DataAPI, webDomain string, data *caseData) (common.ClientView, error) {
	var provider *common.CareProviderAssignment
	for _, assignment := range data.CareTeamMembers {
		if assignment.ProviderID == m.DoctorID {
			provider = assignment
		}
	}

	// It's possible for a CC that's not on the care team to message the patient.
	if provider == nil {
		cc, err := dataAPI.Doctor(m.DoctorID, true)
		if err != nil {
			return nil, errors.Trace(err)
		}
		provider = &common.CareProviderAssignment{
			ShortDisplayName: cc.ShortDisplayName,
			LongDisplayName:  cc.LongDisplayName,
			ProviderID:       cc.ID.Int64(),
			ProviderRole:     api.RoleDoctor,
		}
		if cc.IsCC {
			provider.ProviderRole = api.RoleCC
		}
	}

	if provider == nil {
		return nil, fmt.Errorf("failed to find provider with ID %d for message notification", m.DoctorID)
	}

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("You have a new message from %s.", provider.LongDisplayName),
		IconURL:     app_url.ThumbnailURL(data.APIDomain, provider.ProviderRole, provider.ProviderID),
		ActionURL:   app_url.ViewCaseAction(m.CaseID),
		ButtonTitle: "View Case",
	}

	return nView, nView.Validate()
}

//
// ****************** visit submitted notification ***************/
//

type visitSubmittedNotification struct {
	CaseID  int64 `json:"case_id"`
	VisitID int64 `json:"visit_id"`
}

func (v *visitSubmittedNotification) TypeName() string {
	return CNVisitSubmitted
}

const (
	visitSubmittedTitle = "We'll notify you when your doctor has reviewed your visit."
)

func (v *visitSubmittedNotification) canRenderCaseNotificationView() bool { return false }

func (v *visitSubmittedNotification) makeCaseNotificationView(data *caseData) (common.ClientView, error) {
	return nil, nil
}

func (v *visitSubmittedNotification) makeHomeCardView(dataAPI api.DataAPI, webDomain string, data *caseData) (common.ClientView, error) {
	title := visitSubmittedTitle
	iconURL := app_url.IconCaseLarge.String()
	doctorAssignment := findActiveDoctor(data.CareTeamMembers)

	if doctorAssignment != nil {
		title = fmt.Sprintf("We'll notify you when %s has reviewed your visit.", doctorAssignment.ShortDisplayName)
		iconURL = app_url.ThumbnailURL(data.APIDomain, doctorAssignment.ProviderRole, doctorAssignment.ProviderID)
	}

	nView := &phCaseNotificationStandardView{
		Title:       title,
		IconURL:     iconURL,
		ButtonTitle: "View Case",
		ActionURL:   app_url.ViewCaseAction(v.CaseID),
	}

	return nView, nView.Validate()
}

//
// ********* incomplete visit notification **********
//

type incompleteVisitNotification struct {
	PatientVisitID int64 `json:"PatientVisitId"`
}

func (v *incompleteVisitNotification) TypeName() string {
	return CNIncompleteVisit
}

func (v *incompleteVisitNotification) canRenderCaseNotificationView() bool { return true }

func (v *incompleteVisitNotification) makeCaseNotificationView(data *caseData) (common.ClientView, error) {
	doctorAssignment := findActiveDoctor(data.CareTeamMembers)
	continueVisitMessage := determineContinueVisitMessage(doctorAssignment)

	nView := &caseNotificationTitleSubtitleView{
		Title:     fmt.Sprintf("Continue Your %s Visit", data.Case.Name),
		Subtitle:  continueVisitMessage,
		ID:        data.Notification.ID,
		ActionURL: app_url.ContinueVisitAction(v.PatientVisitID, false),
	}
	return nView, nView.Validate()
}

func (v *incompleteVisitNotification) makeHomeCardView(dataAPI api.DataAPI, webDomain string, data *caseData) (common.ClientView, error) {
	visit, err := dataAPI.GetPatientVisitFromID(v.PatientVisitID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if visit.Status == common.PVStatusPendingParentalConsent {
		actionURL, err := patient.ParentalConsentRequestSMSAction(dataAPI, webDomain, data.Case.PatientID.Int64())
		if err != nil {
			return nil, errors.Trace(err)
		}
		view := &phCaseView{
			Title:     data.Case.Name + " Visit",
			Subtitle:  "Waiting for Parental Consent",
			CaseID:    data.Case.ID.Int64(),
			IconURL:   app_url.IconConsentLarge.String(),
			ActionURL: actionURL,
			NotificationView: &phCaseNotificationStandardView{
				IconURL:     app_url.IconConsent.String(),
				Title:       "A parent needs to consent to your treatment and provide a valid photo ID before you can submit your visit.",
				ButtonTitle: "Send a Reminder",
				ActionURL:   actionURL,
			},
		}
		return view, errors.Trace(view.Validate())
	}

	doctorAssignment := findActiveDoctor(data.CareTeamMembers)

	nView := &phContinueVisit{
		ActionURL: app_url.ContinueVisitAction(v.PatientVisitID, false),
	}
	if doctorAssignment == nil {
		nView.IconURL = app_url.IconCaseLarge.String()
		nView.Subtitle = "With the First Available Doctor"
	} else {
		nView.IconURL = app_url.ThumbnailURL(data.APIDomain, doctorAssignment.ProviderRole, doctorAssignment.ProviderID)
		nView.Subtitle = "With " + doctorAssignment.LongDisplayName
	}
	if visit.Status == common.PVStatusReceivedParentalConsent {
		nView.Title = fmt.Sprintf("Complete Your %s Visit", data.Case.Name)
		nView.Description = "Your parent has provided consent for your visit. Complete your visit to get personalized treatment."
		nView.ButtonTitle = "Complete Visit"
	} else {
		nView.Title = fmt.Sprintf("Continue Your %s Visit", data.Case.Name)
		nView.Description = determineContinueVisitMessage(doctorAssignment)
		nView.ButtonTitle = "Continue Visit"
	}
	return nView, errors.Trace(nView.Validate())
}

func determineContinueVisitMessage(doctorAssignment *common.CareProviderAssignment) string {
	if doctorAssignment != nil {
		return fmt.Sprintf("Complete your visit and get a personalized treatment plan from %s.", doctorAssignment.ShortDisplayName)
	}
	return "Complete your visit and get a personalized treatment plan from your doctor."
}

//
// ***************** incomplete followup visit noficiation ******************
//

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
		ID:            data.Notification.ID,
		Title:         "Complete your follow-up visit",
		IconURL:       app_url.IconCaseSmall,
		ActionURL:     app_url.ContinueVisitAction(v.PatientVisitID, false),
		CreationTime:  data.Notification.CreationDate,
		CreationEpoch: data.Notification.CreationDate.Unix(),
	}
	return nView, nView.Validate()
}

func (v *incompleteFollowupVisitNotification) makeHomeCardView(dataAPI api.DataAPI, webDomain string, data *caseData) (common.ClientView, error) {
	doctorMember := findActiveDoctor(data.CareTeamMembers)

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("Complete your follow-up visit with %s", doctorMember.ShortDisplayName),
		IconURL:     app_url.IconCaseLarge.String(),
		ButtonTitle: "View Case",
		ActionURL:   app_url.ViewCaseAction(v.CaseID),
	}

	return nView, nView.Validate()
}

//
// ***************** start followup visit notification ****************/
//

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
		ID:            data.Notification.ID,
		Title:         "Start your follow-up visit",
		IconURL:       app_url.IconCaseSmall,
		ActionURL:     app_url.ContinueVisitAction(v.PatientVisitID, false),
		CreationTime:  data.Notification.CreationDate,
		CreationEpoch: data.Notification.CreationDate.Unix(),
	}
	return nView, nView.Validate()
}

func (v *startFollowupVisitNotification) makeHomeCardView(dataAPI api.DataAPI, webDomain string, data *caseData) (common.ClientView, error) {
	doctorMember := findActiveDoctor(data.CareTeamMembers)

	nView := &phCaseNotificationStandardView{
		Title:       fmt.Sprintf("%s requested a follow-up visit", doctorMember.ShortDisplayName),
		IconURL:     app_url.ThumbnailURL(data.APIDomain, doctorMember.ProviderRole, doctorMember.ProviderID),
		ButtonTitle: "View Case",
		ActionURL:   app_url.ViewCaseAction(v.CaseID),
	}

	return nView, nView.Validate()
}

//
// ***************** pre submission triage notification ****************/
//
type preSubmissionTriageNotification struct {
	CaseID        int64  `json:"case_id"`
	VisitID       int64  `json:"visit_id"`
	Title         string `json:"title"`
	ActionMessage string `json:"action_message"`
	ActionURL     string `json:"action_url"`
}

func (v *preSubmissionTriageNotification) TypeName() string {
	return CNPreSubmissionTriage
}

func (v *preSubmissionTriageNotification) canRenderCaseNotificationView() bool { return false }

func (v *preSubmissionTriageNotification) makeHomeCardView(dataAPI api.DataAPI, webDomain string, data *caseData) (common.ClientView, error) {
	return &phSectionView{
		Title: v.Title,
		Views: []common.ClientView{
			&phSmallIconText{
				Title:       v.ActionMessage,
				IconURL:     app_url.IconBlueTriage,
				ActionURL:   v.ActionURL,
				RoundedIcon: true,
			},
		},
	}, nil
}

func (v *preSubmissionTriageNotification) makeCaseNotificationView(data *caseData) (common.ClientView, error) {
	return nil, nil
}

func findActiveDoctor(careTeamMembers []*common.CareProviderAssignment) *common.CareProviderAssignment {
	for _, assignment := range careTeamMembers {
		if assignment.Status == api.StatusActive && assignment.ProviderRole == api.RoleDoctor {
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
	registerNotificationType(&preSubmissionTriageNotification{})
}

var NotifyTypes = make(map[string]reflect.Type)

func registerNotificationType(n notification) {
	NotifyTypes[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
}
