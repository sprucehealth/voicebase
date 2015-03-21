package saml

import "strings"

type Intake struct {
	Sections []*Section `yaml:"sections" json:"sections"`
}

type Section struct {
	Title               string        `yaml:"section_title" json:"section_title"`
	TransitionToMessage string        `yaml:"transition_to_message" json:"transition_to_message"`
	Subsections         []*Subsection `yaml:"subsections,omitempty" json:"subsections,omitempty"`
	Screens             []*Screen     `yaml:"screens,omitempty" json:"screens,omitempty"`
}

type Subsection struct {
	Title   string    `yaml:"title" json:"title"`
	Screens []*Screen `yaml:"screens,omitempty" json:"screens,omitempty"`
}

type Screen struct {
	Type       string            `yaml:"screen_type,omitempty" json:"screen_type,omitempty"`
	Title      string            `yaml:"screen_title,omitempty" json:"screen_title,omitempty"`
	Condition  *Condition        `yaml:"condition,omitempty" json:"condition,omitempty"`
	Questions  []*Question       `yaml:"questions,omitempty" json:"questions,omitempty"`
	ClientData *ScreenClientData `yaml:"client_data,omitempty" json:"client_data,omitempty"`
	// Type == "screen_type_photo"
	HeaderTitle    string `yaml:"header_title,omitempty" json:"header_title,omitempty"`
	HeaderSubtitle string `yaml:"header_subtitle,omitempty" json:"header_subtitle,omitempty"`
	HeaderSummary  string `yaml:"header_summary,omitempty" json:"header_summary,omitempty"`
	// Type in ["screen_type_warning_popup", "screen_type_triage"]
	ContentHeaderTitle string      `yaml:"content_header_title,omitempty" json:"content_header_title,omitempty"`
	BottomButtonTitle  string      `yaml:"bottom_button_title,omitempty" json:"bottom_button_title,omitempty"`
	Body               *ScreenBody `yaml:"body,omitempty" json:"body,omitempty"`
}

func (s *Screen) clone() *Screen {
	if s == nil {
		return nil
	}
	s2 := *s
	s2.Condition = s.Condition.clone()
	s2.ClientData = s.ClientData.clone()
	s2.Body = s.Body.clone()
	// TODO: cloning is only currently used for triage so cloning questions isn't necessary
	// s2.Questions = nil
	// for _, q := range s.Questions {
	// 	s2.Questions = append(s2.Questions, q.clone())
	// }
	return &s2
}

type ScreenBody struct {
	Text string `yaml:"text" json:"text"`
}

func (sb *ScreenBody) clone() *ScreenBody {
	if sb == nil {
		return nil
	}
	sb2 := *sb
	return &sb2
}

type ScreenClientData struct {
	PathwayTag                         string        `yaml:"pathway_id,omitempty" json:"pathway_id,omitempty"`
	RequiresAtLeastOneQuestionAnswered *bool         `yaml:"requires_at_least_one_question_answered,omitempty" json:"requires_at_least_one_question_answered,omitempty"`
	Triage                             *TriageParams `yaml:"triage_parameters,omitempty" json:"triage_parameters,omitempty"`
	Views                              []View        `yaml:"views,omitempty" json:"views,omityempty"`
}

func (scd *ScreenClientData) clone() *ScreenClientData {
	if scd == nil {
		return nil
	}
	scd2 := *scd
	scd2.RequiresAtLeastOneQuestionAnswered = cloneBoolPtr(scd.RequiresAtLeastOneQuestionAnswered)
	scd2.Triage = scd.Triage.clone()
	return &scd2
}

type View map[string]interface{}

type Question struct {
	Condition         *Condition                 `yaml:"condition,omitempty" json:"condition,omitempty"`
	Details           *QuestionDetails           `yaml:"details" json:"details"`
	SubquestionConfig *QuestionSubquestionConfig `yaml:"subquestions_config,omitempty" json:"subquestions_config,omitempty"`
}

type QuestionDetails struct {
	Tag              string                    `yaml:"tag,omitempty" json:"tag,omitempty"`
	Text             string                    `yaml:"text" json:"text"`
	Subtext          string                    `yaml:"subtext,omitempty" json:"subtext,omitempty"`
	Summary          string                    `yaml:"summary_text,omitempty" json:"summary_text,omitempty"`
	Type             string                    `yaml:"type" json:"type"`
	AlertText        string                    `yaml:"alert_text,omitempty" json:"alert_text,omitempty"`
	ToAlert          *bool                     `yaml:"to_alert,omitempty" json:"to_alert,omitempty"`
	Global           *bool                     `yaml:"global,omitempty" json:"global,omitempty"`
	Required         *bool                     `yaml:"required,omitempty" json:"required,omitempty"`
	ToPrefill        *bool                     `yaml:"to_prefill,omitempty" json:"to_prefill,omitempty"`
	Answers          []*Answer                 `yaml:"answers,omitempty" json:"answers,omitempty"`
	AdditionalFields *QuestionAdditionalFields `yaml:"additional_question_fields,omitempty" json:"additional_question_fields,omitempty"`
	PhotoSlots       []*PhotoSlot              `yaml:"photo_slots,omitempty" json:"photo_slots,omitempty"`
	AnswerGroups     []*AnswerGroup            `yaml:"answer_groups,omitempty" json:"answer_groups,omitempty"`
}

type AnswerGroup struct {
	Title   string    `yaml:"title" json:"title"`
	Answers []*Answer `yaml:"answers" json:"answers"`
}

type QuestionSubquestionConfig struct {
	Screens   []*Screen   `yaml:"screens,omitempty" json:"screens,omitempty"`
	Questions []*Question `yaml:"questions,omitempty" json:"questions,omitempty"`
}

type QuestionAdditionalFields struct {
	PlaceholderText         string `yaml:"placeholder_text,omitempty" json:"placeholder_text,omitempty"`
	Popup                   *Popup `yaml:"popup,omitempty" json:"popup,omitempty"`
	AllowsMultipleSections  *bool  `yaml:"allows_multiple_sections,omitempty" json:"allows_multiple_sections,omitempty"`
	UserDefinedSectionTitle *bool  `yaml:"user_defined_section_title,omitempty" json:"user_defined_section_title,omitempty"`
	AddButtonText           string `yaml:"add_button_text,omitempty" json:"add_button_text,omitempty"`
	AddText                 string `yaml:"add_text,omitempty" json:"add_text,omitempty"`
	EmptyStateText          string `yaml:"empty_state_text,omitempty" json:"empty_state_text,omitempty"`
	RemoveButtonText        string `yaml:"remove_button_text,omitempty" json:"remove_button_text,omitempty"`
	SaveButtonText          string `yaml:"save_button_text,omitempty" json:"save_button_text,omitempty"`
}

type Popup struct {
	Text string `yaml:"text" json:"text"`
}

type Answer struct {
	Tag        string            `yaml:"tag,omitempty" json:"tag,omitempty"`
	Text       string            `yaml:"text" json:"text"`
	Type       string            `yaml:"type,omitempty" json:"type,omitempty"`
	Summary    string            `yaml:"summary_text,omitempty" json:"summary_text,omitempty"`
	ToAlert    *bool             `yaml:"to_alert,omitempty" json:"to_alert,omitempty"`
	ClientData *AnswerClientData `yaml:"client_data,omitempty" json:"client_data,omitempty"`
}

type AnswerClientData struct {
	PlaceholderText string `yaml:"placeholder_text,omitempty" json:"placeholder_text,omitempty"`
	Popup           *Popup `yaml:"popup,omitempty" json:"popup,omitempty"`
}

type TriageParams struct {
	Title         string `yaml:"title,omitempty" json:"title,omitempty"`
	ActionMessage string `yaml:"action_message,omitempty" json:"action_message,omitempty"`
	ActionURL     string `yaml:"action_url,omitempty" json:"action_url,omitempty"`
	Abandon       *bool  `yaml:"abandon,omitempty" json:"abandon,omitempty"`
}

func (tp *TriageParams) clone() *TriageParams {
	if tp == nil {
		return nil
	}
	tp2 := *tp
	tp2.Abandon = cloneBoolPtr(tp.Abandon)
	return &tp2
}

type PhotoSlot struct {
	Name       string               `yaml:"name" json:"name"`
	Required   *bool                `yaml:"required,omitempty" json:"required,omityempty"`
	ClientData *PhotoSlotClientData `yaml:"client_data,omitempty" json:"client_data,omitempty"`
}

type PhotoSlotClientData struct {
	Tip                      string `yaml:"tip,omitempty" json:"tip,omitempty"`
	TipSubtext               string `yaml:"tip_subtext,omitempty" json:"tip_subtext,omitempty"`
	TipStyle                 string `yaml:"tip_style,omitempty" json:"tip_style,omitempty"`
	OverlayImageURL          string `yaml:"overlay_image_url,omitempty" json:"overlay_image_url,omitempty"`
	PhotoMissingErrorMessage string `yaml:"photo_missing_error_message,omitempty" json:"photo_missing_error_message,omitempty"`
	InitialCameraDirection   string `yaml:"initial_camera_direction,omitempty" json:"initial_camera_direction,omitempty"`
	Flash                    *bool  `yaml:"flash,omitempty" json:"flash,omitempty"`
}

type Condition struct {
	Op               string       `yaml:"op" json:"op"`
	Question         string       `yaml:"question,omitempty" json:"question,omitempty"`
	PotentialAnswers []string     `yaml:"potential_answers,omitempty" json:"potential_answers,omitempty"`
	Operands         []*Condition `yaml:"operands,omitempty" json:"operands,omitempty"`
	Gender           string       `yaml:"gender,omitempty" json:"gender,omitempty"`
}

func (c *Condition) clone() *Condition {
	if c == nil {
		return nil
	}
	c2 := *c
	c2.Operands = nil
	for _, op := range c.Operands {
		c2.Operands = append(c2.Operands, op.clone())
	}
	return &c2
}

func (c *Condition) String() string {
	switch c.Op {
	default:
		return "UNKNOWN-OP-" + c.Op
	case "not":
		return "(NOT " + c.Operands[0].String() + ")"
	case "and":
		s := "("
		for i, o := range c.Operands {
			if i != 0 {
				s += " AND "
			}
			s += o.String()
		}
		return s + ")"
	case "or":
		s := "("
		for i, o := range c.Operands {
			if i != 0 {
				s += " OR "
			}
			s += o.String()
		}
		return s + ")"
	case "answer_contains_any":
		return "(" + c.Question + " any [" + strings.Join(c.PotentialAnswers, ", ") + "])"
	case "answer_contains_all":
		return "(" + c.Question + " all [" + strings.Join(c.PotentialAnswers, ", ") + "])"
	}
}

func cloneBoolPtr(b *bool) *bool {
	if b == nil {
		return nil
	}
	b2 := *b
	return &b2
}