package patient_case

import (
	"time"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

const (
	caseNotificationNameSpace            = "case_notification:"
	patientHomeNameSpace                 = "patient_home:"
	patientHomeCaseNotificationNameSpace = "patient_home_case_notification:"
)

type caseNotificationMessageView struct {
	ID          int64                 `json:"id,string"`
	Type        string                `json:"type"`
	Title       string                `json:"title"`
	IconURL     *app_url.SpruceAsset  `json:"icon_url"`
	ActionURL   *app_url.SpruceAction `json:"action_url"`
	MessageID   int64                 `json:"message_id,string"`
	RoundedIcon bool                  `json:"rounded_icon"`
	DateTime    time.Time             `json:"date_time"`
}

func (c *caseNotificationMessageView) Validate() error {
	c.Type = caseNotificationNameSpace + "message"
	return nil
}

type caseNotificationTitleSubtitleView struct {
	ID        int64                 `json:"id,string"`
	Type      string                `json:"type"`
	Title     string                `json:"title"`
	Subtitle  string                `json:"subtitle"`
	ActionURL *app_url.SpruceAction `json:"action_url,omitempty"`
}

func (c *caseNotificationTitleSubtitleView) Validate() error {
	c.Type = caseNotificationNameSpace + "title_subtitle"
	return nil
}

type phCaseNotificationStandardView struct {
	Type        string                `json:"type"`
	IconURL     *app_url.SpruceAsset  `json:"icon_url"`
	ActionURL   *app_url.SpruceAction `json:"action_url,omitempty"`
	Title       string                `json:"title"`
	Subtitle    string                `json:"subtitle,omitempty"`
	ButtonTitle string                `json:"button_title,omitempty"`
}

func (p *phCaseNotificationStandardView) Validate() error {
	p.Type = patientHomeCaseNotificationNameSpace + "standard"
	return nil
}

type phCaseNotificationMultipleView struct {
	Type              string                `json:"type"`
	NotificationCount int64                 `json:"notification_count"`
	Title             string                `json:"title"`
	ButtonTitle       string                `json:"button_title,omitempty"`
	ActionURL         *app_url.SpruceAction `json:"action_url,omitempty"`
}

func (p *phCaseNotificationMultipleView) Validate() error {
	p.Type = patientHomeCaseNotificationNameSpace + "multiple"
	return nil
}

type phStartVisit struct {
	Type        string                `json:"type"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	ActionURL   *app_url.SpruceAction `json:"action_url"`
	IconURL     *app_url.SpruceAsset  `json:"icon_url"`
	ButtonTitle string                `json:"button_title"`
}

func (p *phStartVisit) Validate() error {
	p.Type = patientHomeNameSpace + "start_visit"
	return nil
}

type phContinueVisit struct {
	Type        string                `json:"type"`
	Title       string                `json:"title"`
	ActionURL   *app_url.SpruceAction `json:"action_url"`
	Description string                `json:"description"`
	ButtonTitle string                `json:"button_title"`
}

func (p *phContinueVisit) Validate() error {
	p.Type = patientHomeNameSpace + "continue_visit"
	return nil
}

type phCaseView struct {
	Type             string                `json:"type"`
	Title            string                `json:"title"`
	Subtitle         string                `json:"subtitle"`
	IconURL          *app_url.SpruceAsset  `json:"icon_url"`
	ActionURL        *app_url.SpruceAction `json:"action_url,omitempty"`
	CaseID           int64                 `json:"case_id,string"`
	NotificationView common.ClientView     `json:"notification_view"`
}

func (p *phCaseView) Validate() error {
	p.Type = patientHomeNameSpace + "case_view"
	if p.NotificationView != nil {
		return p.NotificationView.Validate()
	}

	return nil
}

type phCareProviderView struct {
	Type         string                         `json:"type"`
	CareProvider *common.CareProviderAssignment `json:"care_provider"`
}

func (p *phCareProviderView) Validate() error {
	p.Type = patientHomeNameSpace + "care_provider_view"
	return nil
}

type phSectionView struct {
	Type  string              `json:"type"`
	Title string              `json:"title,omitempty"`
	Views []common.ClientView `json:"views"`
}

func (p *phSectionView) Validate() error {
	p.Type = patientHomeNameSpace + "section"
	for _, view := range p.Views {
		if err := view.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type phSmallIconText struct {
	Type        string                `json:"type"`
	Title       string                `json:"title"`
	Subtitle    string                `json:"subtitle,omitempty"`
	IconURL     *app_url.SpruceAsset  `json:"icon_url"`
	ActionURL   *app_url.SpruceAction `json:"action_url"`
	RoundedIcon bool                  `json:"rounded_icon"`
}

func (p *phSmallIconText) Validate() error {
	p.Type = patientHomeNameSpace + "small_icon_text"
	return nil
}
