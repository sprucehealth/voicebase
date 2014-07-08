package patient_case

import (
	"github.com/sprucehealth/backend/app_url"
)

const (
	caseNotificationNameSpace = "case_notification:"
)

type notificationView interface {
	Validate() error
}

type caseNotificationMessageView struct {
	ID           int64                 `json:"id,string"`
	Type         string                `json:"type"`
	Title        string                `json:"title"`
	IconURL      *app_url.SpruceAsset  `json:"icon_url"`
	ActionURL    *app_url.SpruceAction `json:"action_url"`
	MessageID    int64                 `json:"message_id,string"`
	RoundedIcon  bool                  `json:"rounded_icon"`
	DismissOnTap bool                  `json:"dismiss_on_tap"`
}

func (c *caseNotificationMessageView) Validate() error {
	c.Type = caseNotificationNameSpace + "message"
	return nil
}

type caseNotificationTitleSubtitleView struct {
	ID       int64  `json:"id,string"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

func (c *caseNotificationTitleSubtitleView) Validate() error {
	c.Type = caseNotificationNameSpace + "title_subtitle"
	return nil
}
