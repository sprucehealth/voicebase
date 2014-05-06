package homelog

import "time"

type view interface {
}

type incompleteVisitView struct {
	Type           string `json:"type"`
	Title          string `json:"title"`
	IconURL        string `json:"icon_url"`
	ButtonText     string `json:"button_text"`
	ButtonIconURL  string `json:"button_icon_url,omitempty"`
	TapURL         string `json:"tap_url"`
	PatientVisitId int64  `json:"patient_visit_id,string"`
	NotificationId int64  `json:"notification_id,string"`
}

type messageView struct {
	Dismissible     bool   `json:"dismissible"`
	DismissOnAction bool   `json:"dismiss_on_action"`
	Type            string `json:"type"`
	Title           string `json:"title"`
	IconURL         string `json:"icon_url"`
	ButtonText      string `json:"button_text,omitempty"`
	ButtonIconURL   string `json:"button_icon_url,omitempty"`
	TapURL          string `json:"tap_url"`
	Text            string `json:"text"`
	NotificationId  int64  `json:"notification_id,string"`
}

type bodyButtonView struct {
	Type              string `json:"type"`
	Dismissible       bool   `json:"dismissible"`
	DismissOnAction   bool   `json:"dismiss_on_action"`
	Title             string `json:"title"`
	IconURL           string `json:"icon_url"`
	ButtonText        string `json:"button_text,omitempty"`
	ButtonIconURL     string `json:"button_icon_url,omitempty"`
	TapURL            string `json:"tap_url"`
	BodyButtonIconURL string `json:"body_button_icon_url"`
	BodyButtonText    string `json:"body_button_text"`
	BodyButtonTapURL  string `json:"body_button_url"`
	NotificationId    int64  `json:"notification_id,string"`
}

type titleSubtitleView struct {
	Type     string    `json:"type"`
	DateTime time.Time `json:"date_time"`
	Title    string    `json:"title"`
	Subtitle string    `json:"subtitle"`
	IconURL  string    `json:"icon_url"`
	TapURL   string    `json:"tap_url"`
}

type textView struct {
	Type     string    `json:"type"`
	DateTime time.Time `json:"date_time"`
	Text     string    `json:"text"`
	IconURL  string    `json:"icon_url"`
	TapURL   string    `json:"tap_url"`
}
