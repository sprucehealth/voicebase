package layout_transformer

import (
	"carefront/api"
	"strconv"
)

const (
	EN_LANGUAGE_ID = 1
)

type ClientLayoutProcessor interface {
	TransformIntakeIntoClientLayout(treatment *Treatment) error
}

type ElementProcessor interface {
	FillInDatabaseInfo(dataApi *api.DataService) error
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

func (c *Condition) FillInDatabaseInfo(dataApi *api.DataService) error {
	if c.QuestionTag == "" {
		return nil
	}
	questionId, _, _, err := dataApi.GetQuestionInfo(c.QuestionTag, EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	c.QuestionId = questionId
	c.PotentialAnswersId = make([]string, len(c.PotentialAnswersTags))
	for i, tag := range c.PotentialAnswersTags {
		answerId, _, _, err := dataApi.GetOutcomeInfo(tag, EN_LANGUAGE_ID)
		c.PotentialAnswersId[i] = strconv.Itoa(int(answerId))
		if err != nil {
			return err
		}
	}
	return nil
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

func (t *TipSection) FillInDatabaseInfo(dataApi *api.DataService) error {
	_, tipSectionTitle, tipSectionSubtext, err := dataApi.GetTipSectionInfo(t.TipsSectionTag, EN_LANGUAGE_ID)
	if err != nil {
		return err
	}

	t.TipsSectionTitle = tipSectionTitle
	t.TipsSubtext = tipSectionSubtext

	t.Tips = make([]string, len(t.TipsTags))
	for i, tipTag := range t.TipsTags {
		_, tipText, err := dataApi.GetTipInfo(tipTag, EN_LANGUAGE_ID)
		if err != nil {
			return err
		}
		t.Tips[i] = tipText
	}

	return nil
}

type PotentialOutcome struct {
	ElementProcessor `json:",omitempty"`
	OutcomeId        int64  `json:"potential_outcome_id,string,omitempty"`
	Outcome          string `json:"potential_outcome,omitempty"`
	OutcomeType      string `json:"outcome_type,omitempty"`
}

type Question struct {
	ElementProcessor    `json:",omitempty"`
	QuestionTag         string              `json:"question"`
	QuestionId          int64               `json:"question_id,string,omitempty"`
	QuestionTitle       string              `json:"question_title,omitempty"`
	QuestionType        string              `json:"question_type,omitempty"`
	PotentialAnswerTags []string            `json:"potential_answers"`
	PotentialOutcomes   []*PotentialOutcome `json:"potential_outcomes"`
	ConditionBlock      *Condition          `json:"condition,omitempty"`
	IsMultiSelect       bool                `json:"multiselect,omitempty"`
	Tips                *TipSection         `json:"tips,omitempty"`
}

func (q *Question) FillInDatabaseInfo(dataApi *api.DataService) error {
	questionId, questionTitle, questionType, err := dataApi.GetQuestionInfo(q.QuestionTag, EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	q.QuestionId = questionId
	q.QuestionTitle = questionTitle
	q.QuestionType = questionType

	// go over the potential ansnwer tags to create potential outcome blocks
	q.PotentialOutcomes = make([]*PotentialOutcome, len(q.PotentialAnswerTags))
	for i, answerTag := range q.PotentialAnswerTags {
		outcomeId, outcome, outcomeType, err := dataApi.GetOutcomeInfo(answerTag, EN_LANGUAGE_ID)
		if err != nil {
			return err
		}

		potentialOutcome := &PotentialOutcome{OutcomeId: outcomeId,
			Outcome:     outcome,
			OutcomeType: outcomeType}
		q.PotentialOutcomes[i] = potentialOutcome
	}

	if q.ConditionBlock != nil {
		err := q.ConditionBlock.FillInDatabaseInfo(dataApi)
		if err != nil {
			return err
		}
	}

	if q.Tips != nil {
		err := q.Tips.FillInDatabaseInfo(dataApi)
		if err != nil {
			return err
		}
	}
	return nil
}

type Screen struct {
	ElementProcessor `json:",omitempty"`
	Description      string      `json:"description,omitempty"`
	Questions        []*Question `json:"questions"`
	ScreenType       string      `json:"screen_type,omitempty"`
	ConditionBlock   *Condition  `json:"condition,omitempty"`
}

func (s *Screen) FillInDatabaseInfo(dataApi *api.DataService) error {
	if s.ConditionBlock != nil {
		err := s.ConditionBlock.FillInDatabaseInfo(dataApi)
		if err != nil {
			return err
		}
	}

	if s.Questions != nil {
		for _, question := range s.Questions {
			err := question.FillInDatabaseInfo(dataApi)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type Section struct {
	ElementProcessor `json:",omitempty"`
	SectionTag       string    `json:"section"`
	SectionId        int64     `json:"section_id,string,omitempty"`
	SectionTitle     string    `json:"section_title,omitempty"`
	Screens          []*Screen `json:"screens"`
}

func (s *Section) FillInDatabaseInfo(dataApi *api.DataService) error {
	sectionId, sectionTitle, err := dataApi.GetSectionInfo(s.SectionTag, EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	s.SectionId = sectionId
	s.SectionTitle = sectionTitle
	for _, screen := range s.Screens {
		err := screen.FillInDatabaseInfo(dataApi)
		if err != nil {
			return err
		}
	}
	return nil
}

type Treatment struct {
	ElementProcessor `json:",omitempty"`
	TreatmentTag     string     `json:"treatment"`
	TreatmentId      int64      `json:"treatment_id,string,omitempty"`
	Sections         []*Section `json:"sections"`
}

func (t *Treatment) FillInDatabaseInfo(dataApi *api.DataService) error {
	treatmentId, err := dataApi.GetTreatmentInfo(t.TreatmentTag, EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	t.TreatmentId = treatmentId
	for _, section := range t.Sections {
		err := section.FillInDatabaseInfo(dataApi)
		if err != nil {
			return err
		}
	}
	return nil
}
