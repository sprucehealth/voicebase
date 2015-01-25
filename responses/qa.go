/*
responses is a package intended to represent common internal response subobjects
*/

package responses

import (
	"encoding/json"

	"github.com/sprucehealth/backend/common"
)

type VersionedQuestion struct {
	AlertText                         string                             `json:"alert_text,omitempty"`
	ID                                int64                              `json:"id,string"`
	LanguageID                        int64                              `json:"language_id,string"`
	ParentID                          int64                              `json:"parent_id,string"`
	Required                          bool                               `json:"required"`
	Subtext                           string                             `json:"subtext,omitempty"`
	SummaryText                       string                             `json:"summary_text,omitempty"`
	Tag                               string                             `json:"tag"`
	Text                              string                             `json:"text,omitempty"`
	TextHasTokens                     bool                               `json:"text_has_tokens,string,omitempty"`
	ToAlert                           bool                               `json:"to_alert,string,omitempty"`
	Type                              string                             `json:"type"`
	Version                           int64                              `json:"version,string"`
	VersionedAnswers                  []*VersionedAnswer                 `json:"versioned_answers"`
	VersionedAdditionalQuestionFields *VersionedAdditionalQuestionFields `json:"versioned_additional_question_fields"`
}

func NewVersionedQuestionFromDBModel(dbmodel *common.VersionedQuestion) *VersionedQuestion {
	va := &VersionedQuestion{
		AlertText:     dbmodel.AlertText,
		ID:            dbmodel.ID,
		LanguageID:    dbmodel.LanguageID,
		Subtext:       dbmodel.SubtextText,
		SummaryText:   dbmodel.SummaryText,
		Tag:           dbmodel.QuestionTag,
		Text:          dbmodel.QuestionText,
		TextHasTokens: dbmodel.TextHasTokens,
		ToAlert:       dbmodel.ToAlert,
		Type:          dbmodel.QuestionType,
		Version:       dbmodel.Version,
	}
	if dbmodel.ParentQuestionID != nil {
		va.ParentID = *dbmodel.ParentQuestionID
	}
	return va
}

type VersionedAnswer struct {
	AlertText     string `json:"alert_text,omitempty"`
	ID            int64  `json:"id,string"`
	LanguageID    int64  `json:"language_id,string"`
	Ordering      int64  `json:"ordering,string"`
	QuestionID    int64  `json:"question_id,string"`
	SummaryText   string `json:"summary_text,omitempty"`
	Tag           string `json:"tag"`
	Text          string `json:"text,omitempty"`
	TextHasTokens bool   `json:"text_has_tokens,omitempty"`
	ToAlert       bool   `json:"to_alert,omitempty"`
	Type          string `json:"type"`
	Status        string `json:"status"`
}

func NewVersionedAnswerFromDBModel(dbmodel *common.VersionedAnswer) *VersionedAnswer {
	return &VersionedAnswer{
		ID:          dbmodel.ID,
		LanguageID:  dbmodel.LanguageID,
		Ordering:    dbmodel.Ordering,
		QuestionID:  dbmodel.QuestionID,
		SummaryText: dbmodel.AnswerSummaryText,
		Tag:         dbmodel.AnswerTag,
		Text:        dbmodel.AnswerText,
		ToAlert:     dbmodel.ToAlert,
		Type:        dbmodel.AnswerType,
		Status:      dbmodel.Status,
	}
}

type VersionedAdditionalQuestionFields map[string]interface{}

func VersionedAdditionalQuestionFieldsFromDBModels(dbmodels []*common.VersionedAdditionalQuestionField) (*VersionedAdditionalQuestionFields, error) {
	var vaqf VersionedAdditionalQuestionFields = make(map[string]interface{})
	for _, field := range dbmodels {
		var jsonMap map[string]interface{}
		if err := json.Unmarshal(field.JSON, &jsonMap); err != nil {
			return nil, err
		}
		for k, v := range jsonMap {
			vaqf[k] = v
		}
	}
	return &vaqf, nil
}
