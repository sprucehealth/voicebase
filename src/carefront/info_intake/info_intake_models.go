package info_intake

import (
	"carefront/api"
	"carefront/common"
	"time"
)

const (
	FORMATTED_FIELD_DOCTOR_LAST_NAME = "doctor_last_name"
	FORMATTED_TITLE_FIELD            = "title"
	QUESTION_TYPE_AUTOCOMPLETE       = "q_type_autocomplete"
	QUESTION_TYPE_COMPOUND           = "q_type_compound"
	QUESTION_TYPE_FREE_TEXT          = "q_type_free_text"
	QUESTION_TYPE_MULTIPLE_CHOICE    = "q_type_multiple_choice"
	QUESTION_TYPE_MULTIPLE_PHOTO     = "q_type_multiple_photo"
	QUESTION_TYPE_PHOTO              = "q_type_photo"
	QUESTION_TYPE_SEGMENTED_CONTROL  = "q_type_segmented_control"
	QUESTION_TYPE_SINGLE_ENTRY       = "q_type_single_entry"
	QUESTION_TYPE_SINGLE_PHOTO       = "q_type_single_photo"
	QUESTION_TYPE_SINGLE_SELECT      = "q_type_single_select"
)

type InfoIntakeModel interface {
	FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error
	GetHealthConditionTag() string
}

type Condition struct {
	OperationTag         string   `json:"op,omitempty"`
	IsServerCondition    bool     `json:"server_condition,omitempty"`
	GenderField          string   `json:"gender,omitempty"`
	QuestionTag          string   `json:"question,omitempty"`
	QuestionId           int64    `json:"question_id,string,omitempty"`
	PotentialAnswersId   []string `json:"potential_answers_id,omitempty"`
	PotentialAnswersTags []string `json:"potential_answers,omitempty"`
	FieldTag             string   `json:"field,omitempty"`
	ValueTag             string   `json:"value,omitempty"`
}

type TipSection struct {
	TipsSectionTag   string   `json:"tips_section_tag"`
	TipsSectionTitle string   `json:"tips_section_title,omitempty"`
	TipsSubtext      string   `json:"tips_subtext, omitempty"`
	PhotoTipsTags    []string `json:"photo_tips,omitempty"`
	TipsTags         []string `json:"tips"`
	Tips             []string `json:"tips_text"`
}

type PotentialAnswer struct {
	AnswerId      int64    `json:"potential_answer_id,string,omitempty"`
	Answer        string   `json:"potential_answer,omitempty"`
	AnswerSummary string   `json:"potential_answer_summary,omitempty"`
	AnswerTypes   []string `json:"answer_types,omitempty"`
	Ordering      int64    `json:"ordering,string"`
	ToAlert       bool     `json:"to_alert"`
}

type Question struct {
	QuestionTag           string                 `json:"question"`
	QuestionId            int64                  `json:"question_id,string,omitempty"`
	QuestionTitle         string                 `json:"question_title,omitempty"`
	QuestionTypes         []string               `json:"question_types,omitempty"`
	FormattedFieldTags    []string               `json:"formatted_field_tags,omitempty"`
	QuestionSubText       string                 `json:"question_subtext,omitempty"`
	QuestionSummary       string                 `json:"question_summary,omitempty"`
	AdditionalFields      map[string]string      `json:"additional_fields,omitempty"`
	DisplayStyles         []string               `json:"display_styles,omitempty"`
	ParentQuestionId      int64                  `json:"parent_question_id,string,omitempty"`
	PotentialAnswers      []*PotentialAnswer     `json:"potential_answers,omitempty"`
	Answers               []*common.AnswerIntake `json:"answers,omitempty"`
	Questions             []*Question            `json:"questions,omitempty"`
	ConditionBlock        *Condition             `json:"condition,omitempty"`
	Tips                  *TipSection            `json:"tips,omitempty"`
	Required              bool                   `json:"required"`
	ToAlert               bool                   `json:"to_alert"`
	AlertFormattedText    string                 `json:"alert_text"`
	PhotoSlots            []*PhotoSlot           `json:"photo_slots,omitempty"`
	AnsweredPhotoSections []*PhotoIntakeSection  `json:"answered_photo_sections,omitempty"`
}

type PhotoSlot struct {
	Id       int64  `json:"id,string"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type PhotoIntakeSection struct {
	Id     int64              `json:"-"`
	Name   string             `json:"name,omitempty"`
	Photos []*PhotoIntakeSlot `json:"photos,omitempty"`
}

type PhotoIntakeSlot struct {
	Id           int64     `json:"-"`
	CreationDate time.Time `json:"creation_date"`
	PhotoUrl     string    `json:"photo_url"`
	SlotId       int64     `json:"slot_id"`
}

type Screen struct {
	Description    string      `json:"description,omitempty"`
	Questions      []*Question `json:"questions"`
	ScreenType     string      `json:"screen_type,omitempty"`
	ConditionBlock *Condition  `json:"condition,omitempty"`
}

type Section struct {
	SectionTag        string      `json:"section"`
	SectionId         int64       `json:"section_id,string,omitempty"`
	SectionTitle      string      `json:"section_title,omitempty"`
	Questions         []*Question `json:"questions,omitempty"`
	Screens           []*Screen   `json:"screens,omitempty"`
	SectionTransition *Transition `json:"transition,omitempty"`
}

type Transition struct {
	Title    string    `json:"title"`
	Message  string    `json:"message,omitempty"`
	ImageUrl string    `json:"image_url,omitempty"`
	Buttons  []*Button `json:"buttons,omitempty"`
}

type Button struct {
	Text   string `json:"button_text,omitempty"`
	TapUrl string `json:"tap_url,omitempty"`
	Style  string `json:"style,omitempty"`
}

type InfoIntakeLayout struct {
	HealthConditionTag string     `json:"health_condition"`
	HealthConditionId  int64      `json:"health_condition_id,string,omitempty"`
	Sections           []*Section `json:"sections"`
}
