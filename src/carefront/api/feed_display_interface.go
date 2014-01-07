package api

const (
	QUEUE_ITEM_STATUS_PENDING                 = "PENDING"
	QUEUE_ITEM_STATUS_COMPLETED               = "COMPLETED"
	QUEUE_ITEM_STATUS_ONGOING                 = "ONGOING"
	DISPLAY_TYPE_TITLE_SUBTITLE_NONACTIONABLE = "title_subtitle_nonactionable"
	DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE    = "title_subtitle_actionable"
	DISPLAY_TYPE_TITLE_SUBTITLE_BUTTON        = "title_subtitle_text_button_actionable"
)

type Button struct {
	ButtonText      string `json:"text"`
	ButtonActionUrl string `json:"action_url"`
}

type DoctorFeedDisplayInterface interface {
	GetTitleAndSubtitle(dataApi DataAPI) (title, subtitle string, err error)
	GetImageTag() string
	GetDisplayTypes() []string
	GetButton() *Button
}
