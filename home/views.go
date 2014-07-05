package home

import "github.com/sprucehealth/backend/app_url"

const (
	patientHomeNameSpace = "patient_home:"
)

type PHView interface {
	Validate() error
}

type PHPrimaryActionView struct {
	Type        string                `json:"type"`
	Title       string                `json:"title"`
	ActionURL   *app_url.SpruceAction `json:"action_url"`
	ButtonTitle string                `json:"button_title"`
}

func (p *PHPrimaryActionView) Validate() error {
	p.Type = patientHomeNameSpace + "primary_action"
	return nil
}

type PHCaseView struct {
	Type             string                `json:"type"`
	Title            string                `json:"title"`
	Subtitle         string                `json:"subtitle"`
	ActionURL        *app_url.SpruceAction `json:"action_url"`
	NotificationView PHView                `json:"notification_view"`
}

func (p *PHCaseView) Validate() error {
	p.Type = patientHomeNameSpace + "case_view"
	if p.NotificationView != nil {
		return p.NotificationView.Validate()
	}

	return nil
}

type PHSmallIconText struct {
	Type        string                `json:"type"`
	Title       string                `json:"title"`
	IconURL     *app_url.SpruceAsset  `json:"icon_url"`
	ActionURL   *app_url.SpruceAction `json:"action_url"`
	RoundedIcon bool                  `json:"rounded_icon"`
}

func (p *PHSmallIconText) Validate() error {
	p.Type = patientHomeNameSpace + "small_icon_text"
	return nil
}
