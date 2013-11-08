package api

type ClientLayoutProcessor interface {
	TransformIntakeIntoClientLayout(treatment *Treatment, languageId int64) error
}

type ElementProcessor interface {
	FillInDatabaseInfo(dataApi DataAPI, languageId int64) error
}

type Condition struct {
	ElementProcessor     `json:",omitempty"`
	OperationTag         string   `json:"op",omitempty`
	IsServerCondition    bool     `json:"server_condition,omitempty"`
	QuestionTag          string   `json:"question,omitempty"`
	QuestionId           int64    `json:"question_id,string,omitempty"`
	PotentialAnswersId   []string `json:"potential_answers_id,omitempty"`
	PotentialAnswersTags []string `json:"potential_answers,omitempty"`
	FieldTag             string   `json:"field,omitempty"`
	ValueTag             string   `json:"value,omitempty"`
}

type TipSection struct {
	ElementProcessor `json:",omitempty"`
	TipsSectionTag   string   `json:"tips_section_tag"`
	TipsSectionTitle string   `json:"tips_section_title,omitempty"`
	TipsSubtext      string   `json:"tips_subtext, omitempty"`
	PhotoTipsTags    []string `json:"photo_tips,omitempty"`
	TipsTags         []string `json:"tips"`
	Tips             []string `json:"tips_text"`
}

type PotentialOutcome struct {
	ElementProcessor `json:",omitempty"`
	OutcomeId        int64  `json:"potential_outcome_id,string,omitempty"`
	Outcome          string `json:"potential_outcome,omitempty"`
	OutcomeType      string `json:"outcome_type,omitempty"`
	Ordering         int64  `json:"ordering"`
}

type Question struct {
	ElementProcessor  `json:",omitempty"`
	QuestionTag       string              `json:"question"`
	QuestionId        int64               `json:"question_id,string,omitempty"`
	QuestionTitle     string              `json:"question_title,omitempty"`
	QuestionType      string              `json:"question_type,omitempty"`
	PotentialOutcomes []*PotentialOutcome `json:"potential_outcomes"`
	ConditionBlock    *Condition          `json:"condition,omitempty"`
	Tips              *TipSection         `json:"tips,omitempty"`
}

type Screen struct {
	ElementProcessor `json:",omitempty"`
	Description      string      `json:"description,omitempty"`
	Questions        []*Question `json:"questions"`
	ScreenType       string      `json:"screen_type,omitempty"`
	ConditionBlock   *Condition  `json:"condition,omitempty"`
}

type Section struct {
	ElementProcessor `json:",omitempty"`
	SectionTag       string    `json:"section"`
	SectionId        int64     `json:"section_id,string,omitempty"`
	SectionTitle     string    `json:"section_title,omitempty"`
	Screens          []*Screen `json:"screens"`
}

type Treatment struct {
	ElementProcessor `json:",omitempty"`
	TreatmentTag     string     `json:"treatment"`
	TreatmentId      int64      `json:"treatment_id,string,omitempty"`
	Sections         []*Section `json:"sections"`
}
