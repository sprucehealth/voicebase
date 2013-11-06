package layout_transformer

type Condition struct {
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
	TipsSectionTag   string   `json:"tips_section_tag"`
	TipsSectionTitle string   `json:"tips_section_title,omitempty"`
	TipsSubtext      string   `json:"tips_subtext, omitempty"`
	PhotoTipsTags    []string `json:"photo_tips,omitempty"`
	TipsTags         []string `json:"tips"`
	Tips             []string `json:"tips_text"`
}

type PotentialOutcome struct {
	OutcomeId   int64  `json:"potential_outcome_id,string,omitempty"`
	Outcome     string `json:"potential_outcome,omitempty"`
	OutcomeType string `json:"outcome_type,omitempty"`
}

type Question struct {
	QuestionTag         string             `json:"question"`
	QuestionId          int64              `json:"question_id,string,omitempty"`
	QuestionTitle       string             `json:"question_title,omitempty"`
	QuestionType        string             `json:"question_type,omitempty"`
	PotentialAnswerTags []string           `json:"potential_answers"`
	PotentialOutcomes   []PotentialOutcome `json:"potential_outcomes"`
	ConditionBlock      *Condition         `json:"condition,omitempty"`
	IsMultiSelect       bool               `json:"multiselect,omitempty"`
	Tips                *TipSection        `json:"tips,omitempty"`
}

type Screen struct {
	Description    string     `json:"description,omitempty"`
	Questions      []Question `json:"questions"`
	ScreenType     string     `json:"screen_type,omitempty"`
	ConditionBlock *Condition `json:"condition,omitempty"`
}

type Section struct {
	SectionTag   string   `json:"section"`
	SectionId    int64    `json:"section_id,string,omitempty"`
	SectionTitle string   `json:"section_title,omitempty"`
	Screens      []Screen `json:"screens"`
}

type Treatment struct {
	TreatmentTag string    `json:"treatment"`
	TreatmentId  int64     `json:"treatment_id,string,omitempty"`
	Sections     []Section `json:"sections"`
}
