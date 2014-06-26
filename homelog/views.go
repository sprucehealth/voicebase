package homelog

import (
	"github.com/sprucehealth/backend/app_url"
	"time"
)

type view interface {
}

type incompleteVisitView struct {
	Type           string                `json:"type"`
	Title          string                `json:"title"`
	IconURL        *app_url.SpruceAsset  `json:"icon_url"`
	ButtonText     string                `json:"button_text"`
	ButtonIconURL  *app_url.SpruceAsset  `json:"button_icon_url,omitempty"`
	TapURL         *app_url.SpruceAction `json:"tap_url"`
	PatientVisitId int64                 `json:"patient_visit_id,string"`
	NotificationId int64                 `json:"notification_id,string"`
}

type messageView struct {
	Dismissible     bool                  `json:"dismissible"`
	DismissOnAction bool                  `json:"dismiss_on_action"`
	Type            string                `json:"type"`
	Title           string                `json:"title"`
	IconURL         *app_url.SpruceAsset  `json:"icon_url"`
	ButtonText      string                `json:"button_text,omitempty"`
	ButtonIconURL   *app_url.SpruceAsset  `json:"button_icon_url,omitempty"`
	TapURL          *app_url.SpruceAction `json:"tap_url"`
	Text            string                `json:"text"`
	NotificationId  int64                 `json:"notification_id,string"`
}

type bodyButtonView struct {
	Type              string                `json:"type"`
	Dismissible       bool                  `json:"dismissible"`
	DismissOnAction   bool                  `json:"dismiss_on_action"`
	Title             string                `json:"title"`
	IconURL           *app_url.SpruceAsset  `json:"icon_url"`
	ButtonText        string                `json:"button_text,omitempty"`
	ButtonIconURL     *app_url.SpruceAsset  `json:"button_icon_url,omitempty"`
	TapURL            *app_url.SpruceAction `json:"tap_url"`
	BodyButtonIconURL *app_url.SpruceAsset  `json:"body_button_icon_url"`
	BodyButtonText    string                `json:"body_button_text"`
	BodyButtonTapURL  *app_url.SpruceAction `json:"body_button_url"`
	NotificationId    int64                 `json:"notification_id,string"`
}

type titleSubtitleView struct {
	Type     string                `json:"type"`
	DateTime time.Time             `json:"date_time"`
	Title    string                `json:"title"`
	Subtitle string                `json:"subtitle"`
	IconURL  *app_url.SpruceAsset  `json:"icon_url"`
	TapURL   *app_url.SpruceAction `json:"tap_url"`
}

type textView struct {
	Type     string                `json:"type"`
	DateTime time.Time             `json:"date_time"`
	Text     string                `json:"text"`
	IconURL  *app_url.SpruceAsset  `json:"icon_url"`
	TapURL   *app_url.SpruceAction `json:"tap_url"`
}
