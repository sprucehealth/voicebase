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
	ParentID                          int64                              `json:"parent_id,string,omitempty"`
	Required                          bool                               `json:"required"`
	Subtext                           string                             `json:"subtext,omitempty"`
	SummaryText                       string                             `json:"summary_text,omitempty"`
	Tag                               string                             `json:"tag"`
	Text                              string                             `json:"text,omitempty"`
	TextHasTokens                     bool                               `json:"text_has_tokens,omitempty"`
	ToAlert                           bool                               `json:"to_alert,omitempty"`
	Type                              string                             `json:"type"`
	Version                           int64                              `json:"version,string"`
	VersionedAnswers                  []*VersionedAnswer                 `json:"versioned_answers"`
	VersionedAdditionalQuestionFields *VersionedAdditionalQuestionFields `json:"versioned_additional_question_fields"`
	VersionedPhotoSlots               []*VersionedPhotoSlot              `json:"versioned_photo_slots"`
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
		Required:      dbmodel.Required,
	}
	if dbmodel.ParentQuestionID != nil {
		va.ParentID = *dbmodel.ParentQuestionID
	}
	return va
}

type VersionedAnswer struct {
	AlertText     string                 `json:"alert_text,omitempty"`
	ID            int64                  `json:"id,string"`
	LanguageID    int64                  `json:"language_id,string"`
	Ordering      int64                  `json:"ordering,string"`
	QuestionID    int64                  `json:"question_id,string"`
	SummaryText   string                 `json:"summary_text,omitempty"`
	Tag           string                 `json:"tag"`
	Text          string                 `json:"text,omitempty"`
	TextHasTokens bool                   `json:"text_has_tokens,omitempty"`
	ToAlert       bool                   `json:"to_alert,omitempty"`
	Type          string                 `json:"type"`
	Status        string                 `json:"status"`
	ClientData    map[string]interface{} `json:"client_data"`
}

func NewVersionedAnswerFromDBModel(dbmodel *common.VersionedAnswer) (*VersionedAnswer, error) {
	var clientData map[string]interface{}
	if dbmodel.ClientData != nil {
		if err := json.Unmarshal(dbmodel.ClientData, &clientData); err != nil {
			return nil, err
		}
	}

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
		ClientData:  clientData,
	}, nil
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

type VersionedPhotoSlot struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Required   bool                   `json:"required"`
	Status     string                 `json:"status"`
	Ordering   int64                  `json:"ordering,string"`
	LanguageID int64                  `json:"language_id,string"`
	ClientData map[string]interface{} `json:"client_data"`
	QuestionID int64                  `json:"question_id,string"`
	ID         int64                  `json:"id,string"`
}

func NewVersionedPhotoSlotFromDBModel(dbmodel *common.VersionedPhotoSlot) (*VersionedPhotoSlot, error) {
	var clientData map[string]interface{}
	if len(dbmodel.ClientData) > 0 {
		if err := json.Unmarshal(dbmodel.ClientData, &clientData); err != nil {
			return nil, err
		}
	}
	return &VersionedPhotoSlot{
		LanguageID: dbmodel.LanguageID,
		Ordering:   dbmodel.Ordering,
		QuestionID: dbmodel.QuestionID,
		Name:       dbmodel.Name,
		Type:       dbmodel.Type,
		Status:     dbmodel.Status,
		ID:         dbmodel.ID,
		Required:   dbmodel.Required,
		ClientData: clientData,
	}, nil
}
