package homelog

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
}

type bodyButtonView struct {
	Dismissible       bool   `json:"dismissible"`
	DismissOnAction   bool   `json:"dismiss_on_action"`
	Type              string `json:"type"`
	Title             string `json:"title"`
	IconURL           string `json:"icon_url"`
	ButtonText        string `json:"button_text,omitempty"`
	ButtonIconURL     string `json:"button_icon_url,omitempty"`
	TapURL            string `json:"tap_url"`
	BodyButtonIconURL string `json:"body_button_icon_url"`
	BodyButtonText    string `json:"body_button_text"`
	BodyButtonTapURL  string `json:"body_button_url"`
}
