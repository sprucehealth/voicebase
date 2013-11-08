package info_intake

import (
	"carefront/api"
)

type InfoIntakeModelFiller interface {
	FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error
}

type Condition struct {
	InfoIntakeModelFiller `json:",omitempty"`
	OperationTag          string   `json:"op",omitempty`
	IsServerCondition     bool     `json:"server_condition,omitempty"`
	QuestionTag           string   `json:"question,omitempty"`
	QuestionId            int64    `json:"question_id,string,omitempty"`
	PotentialAnswersId    []string `json:"potential_answers_id,omitempty"`
	PotentialAnswersTags  []string `json:"potential_answers,omitempty"`
	FieldTag              string   `json:"field,omitempty"`
	ValueTag              string   `json:"value,omitempty"`
}

type TipSection struct {
	InfoIntakeModelFiller `json:",omitempty"`
	TipsSectionTag        string   `json:"tips_section_tag"`
	TipsSectionTitle      string   `json:"tips_section_title,omitempty"`
	TipsSubtext           string   `json:"tips_subtext, omitempty"`
	PhotoTipsTags         []string `json:"photo_tips,omitempty"`
	TipsTags              []string `json:"tips"`
	Tips                  []string `json:"tips_text"`
}

type PotentialOutcome struct {
	InfoIntakeModelFiller `json:",omitempty"`
	OutcomeId             int64  `json:"potential_outcome_id,string,omitempty"`
	Outcome               string `json:"potential_outcome,omitempty"`
	OutcomeType           string `json:"outcome_type,omitempty"`
	Ordering              int64  `json:"ordering"`
}

type Question struct {
	InfoIntakeModelFiller `json:",omitempty"`
	QuestionTag           string              `json:"question"`
	QuestionId            int64               `json:"question_id,string,omitempty"`
	QuestionTitle         string              `json:"question_title,omitempty"`
	QuestionType          string              `json:"question_type,omitempty"`
	PotentialOutcomes     []*PotentialOutcome `json:"potential_outcomes"`
	ConditionBlock        *Condition          `json:"condition,omitempty"`
	Tips                  *TipSection         `json:"tips,omitempty"`
}

type Screen struct {
	InfoIntakeModelFiller `json:",omitempty"`
	Description           string      `json:"description,omitempty"`
	Questions             []*Question `json:"questions"`
	ScreenType            string      `json:"screen_type,omitempty"`
	ConditionBlock        *Condition  `json:"condition,omitempty"`
}

type Section struct {
	InfoIntakeModelFiller `json:",omitempty"`
	SectionTag            string    `json:"section"`
	SectionId             int64     `json:"section_id,string,omitempty"`
	SectionTitle          string    `json:"section_title,omitempty"`
	Screens               []*Screen `json:"screens"`
}

type HealthCondition struct {
	InfoIntakeModelFiller `json:",omitempty"`
	HealthConditionTag    string     `json:"health_condition"`
	HealthConditionId     int64      `json:"health_condition_id,string,omitempty"`
	Sections              []*Section `json:"sections"`
}
