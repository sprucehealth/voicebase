/*
responses is a package intended to represent common internal response subobjects
*/

package responses

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/common"
)

// An entry representing an individual care team member
type PatientCareTeamMember struct {
	ProviderRole      string    `json:"provider_role"`
	ProviderID        int64     `json:"provider_id,string"`
	FirstName         string    `json:"first_name,omitempty"`
	LastName          string    `json:"last_name,omitempty"`
	ShortTitle        string    `json:"short_title,omitempty"`
	LongTitle         string    `json:"long_title,omitempty"`
	ShortDisplayName  string    `json:"short_display_name,omitempty"`
	LongDisplayName   string    `json:"long_display_name,omitempty"`
	SmallThumbnailURL string    `json:"small_thumbnail_url,omitempty"`
	LargeThumbnailURL string    `json:"large_thumbnail_url,omitempty"`
	CreationDate      time.Time `json:"assignment_date"`
}

func (p *PatientCareTeamMember) String() string {
	return fmt.Sprintf("{ProviderID: %v, ProviderRole: %v, CreationDate: %v}", p.ProviderID, p.ProviderRole, p.CreationDate)
}

// A summary object representing an individual care team
type PatientCareTeamSummary struct {
	CaseID  int64                    `json:"case_id,string"`
	Members []*PatientCareTeamMember `json:"members"`
}

func (p *PatientCareTeamSummary) String() string {
	return fmt.Sprintf("{CaseID: %v, Members: %v}", p.CaseID, p.Members)
}

// An object representing a chief complaint with localization fields
type ChiefComplaint struct {
	ID            int64  `json:"id,string"`
	Name          string `json:"name,omitempty"`
	NameLocalized string `json:"name_localized,omitempty"`
}

func (c *ChiefComplaint) String() string {
	return fmt.Sprintf("{ID: %v, Name: %v, NameLocalized: %v}", c.ID, c.Name, c.NameLocalized)
}

// An object representing a case with a chief complaint
type Case struct {
	ID             int64           `json:"id,string"`
	ChiefComplaint *ChiefComplaint `json:"chief_complaint,omitempty"`
}

func (c *Case) String() string {
	return fmt.Sprintf("{ID: %v, ChiefComplaint: %v}", c.ID, c.ChiefComplaint)
}

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
