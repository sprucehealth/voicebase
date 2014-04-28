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
