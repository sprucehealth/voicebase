package layout

import "fmt"

const (
	QuestionTypePhotoSection = "q_type_photo_section"
)

// Intake is a layout object that the patient app
// consumes to display the visit to a patient for intake purposes.
type Intake struct {
	Header      *Header           `json:"visit_overview_header,omitempty"`
	Transitions []*TransitionItem `json:"transitions,omitempty"`
	Sections    []*Section        `json:"sections"`
}

type Header struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	IconURL  string `json:"icon_url,omitempty"`
}

type TransitionItem struct {
	Message string    `json:"message"`
	Buttons []*Button `json:"buttons"`
}

type Section struct {
	ID      string    `json:"id"`
	Title   string    `json:"title,omitempty"`
	Screens []*Screen `json:"screens,omitempty"`
}

type Button struct {
	Text   string `json:"button_text,omitempty"`
	TapURL string `json:"tap_url,omitempty"`
	Style  string `json:"style,omitempty"`
}

type Body struct {
	Text   string  `json:"text,omitempty"`
	Button *Button `json:"button,omitempty"`
}

type Condition struct {
	Operation          string       `json:"op,omitempty"`
	GenderField        string       `json:"gender,omitempty"`
	QuestionID         string       `json:"question_id,omitempty"`
	PotentialAnswersID []string     `json:"potential_answers_id,omitempty"`
	Operands           []*Condition `json:"operands,omitempty"`
}

type Screen struct {
	ID                   string            `json:"id"`
	HeaderTitle          string            `json:"header_title,omitempty"`
	HeaderTitleHasTokens bool              `json:"header_title_has_tokens"`
	HeaderSubtitle       string            `json:"header_subtitle,omitempty"`
	HeaderSummary        string            `json:"header_summary,omitempty"`
	Questions            []*Question       `json:"questions,omitempty"`
	Type                 string            `json:"screen_type,omitempty"`
	Condition            *Condition        `json:"condition,omitempty"`
	Body                 *Body             `json:"body,omitempty"`
	BottomButtonTitle    string            `json:"bottom_button_title,omitempty"`
	ContentTitle         string            `json:"content_header_title,omitempty"`
	Title                string            `json:"screen_title,omitempty"`
	ClientData           *ScreenClientData `json:"client_data,omitempty"`
}

type ScreenClientData struct {
	RequiresAtLeastOneQuestionAnswered *bool                    `json:"requires_at_least_one_question_answered,omitempty"`
	Triage                             *TriageParams            `json:"triage_params,omitempty"`
	Views                              []map[string]interface{} `json:"views,omitempty"`
}

type TriageParams struct {
	Title         string `json:"title,omitempty"`
	ActionMessage string `json:"action_message"`
	ActionURL     string `json:"action_url"`
	Abandon       *bool  `json:"abandon,omitempty"`
}

type Question struct {
	ID                 string                    `json:"id,omitempty"`
	Title              string                    `json:"question_title,omitempty"`
	TitleHasTokens     bool                      `json:"question_title_has_tokens"`
	Type               string                    `json:"type,omitempty"`
	Subtext            string                    `json:"question_subtext,omitempty"`
	Summary            string                    `json:"question_summary,omitempty"`
	AdditionalFields   *QuestionAdditionalFields `json:"additional_fields,omitempty"`
	ParentQuestionID   int64                     `json:"parent_question_id,string,omitempty"`
	PotentialAnswers   []*PotentialAnswer        `json:"potential_answers,omitempty"`
	Condition          *Condition                `json:"condition,omitempty"`
	Required           *bool                     `json:"required,omitempty"`
	AlertFormattedText string                    `json:"alert_text,omitempty"`
	PhotoSlots         []*PhotoSlot              `json:"photo_slots,omitempty"`
	SubQuestionsConfig *SubQuestionsConfig       `json:"subquestions_config,omitempty"`
	ToAlert            *bool                     `json:"to_alert,omitempty"`
}

type QuestionAdditionalFields struct {
	PlaceholderText         string         `json:"placeholder_text,omitempty"`
	Popup                   *Popup         `json:"popup,omitempty"`
	AllowsMultipleSections  *bool          `json:"allows_multiple_sections,omitempty"`
	UserDefinedSectionTitle *bool          `json:"user_defined_section_title,omitempty"`
	AddButtonText           string         `json:"add_button_text,omitempty"`
	AddText                 string         `json:"add_text,omitempty"`
	EmptyStateText          string         `json:"empty_state_text,omitempty"`
	RemoveButtonText        string         `json:"remove_button_text,omitempty"`
	SaveButtonText          string         `json:"save_button_text,omitempty"`
	AnswerGroups            []*AnswerGroup `json:"answer_groups,omitempty"`
}

type AnswerGroup struct {
	Count int    `json:"count"`
	Title string `json:"title"`
}

type PotentialAnswer struct {
	ID         string            `json:"id,omitempty"`
	Answer     string            `json:"potential_answer,omitempty"`
	Summary    string            `json:"potential_answer_summary,omitempty"`
	Type       string            `json:"type,omitempty"`
	ToAlert    *bool             `json:"to_alert,omitempty"`
	ClientData *AnswerClientData `json:"client_data,omitempty"`
}

type AnswerClientData struct {
	PlaceholderText string `json:"placeholder_text,omitempty"`
	Popup           *Popup `json:"popup"`
}

type Popup struct {
	Text string `json:"text"`
}

type SubQuestionsConfig struct {
	Screens []*Screen `json:"screens,omitempty"`
}

type PhotoSlot struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	Required   *bool                `json:"required,omitempty"`
	ClientData *PhotoSlotClientData `json:"client_data"`
}

type PhotoSlotClientData struct {
	PhotoTip
	OverlayImageURL          string               `json:"overlay_image_url,omitempty"`
	PhotoMissingErrorMessage string               `json:"photo_missing_error_message,omitempty"`
	InitialCameraDirection   string               `json:"initial_camera_direction,omitempty"`
	Flash                    FlashState           `json:"flash,omitempty"`
	Tips                     map[string]*PhotoTip `json:"tips,omitempty"`
}

type PhotoTip struct {
	Tip        string `json:"tip,omitempty"`
	TipSubtext string `json:"tip_subtext,omitempty"`
	TipStyle   string `json:"tip_style,omitempty"`
}

type FlashState string

const (
	FlashOff  FlashState = "off"
	FlashOn   FlashState = "on"
	FlashAuto FlashState = "auto"
)

func ParseFlashState(str string) (FlashState, error) {
	switch fs := FlashState(str); fs {
	case FlashState(""):
		return fs, nil
	case FlashOff, FlashOn, FlashAuto:
		return fs, nil
	}

	return FlashState(""), fmt.Errorf("Unknown flash state %s", str)
}
