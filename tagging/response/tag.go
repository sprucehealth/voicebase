package response

import "github.com/sprucehealth/backend/tagging/model"

type AssociationDataType string

var (
	CaseAssociationType AssociationDataType = "case"
)

type AssociationDescription interface{}

type TagMembership struct {
	TagID       int64  `json:"tag_id,string"`
	CaseID      *int64 `json:"case_id,string"`
	TriggerTime *int64 `json:"trigger_time"`
	Created     int64  `json:"created"`
	Hidden      bool   `json:"hidden"`
}

type TagAssociation struct {
	ID          int64                  `json:"id,string"`
	Description AssociationDescription `json:"description"`
	Type        AssociationDataType    `json:"type"`
}

type PHISafeCaseAssociationDescription struct {
	PatientInitials      string  `json:"patient_initials"`
	Pathway              string  `json:"pathway"`
	VisitSubmittedEpochs []int64 `json:"visit_submitted_epochs"`
}

type CaseAssociationDescription struct {
	PatientFirstName     string  `json:"patient_first_name"`
	PatientLastName      string  `json:"patient_last_name"`
	Pathway              string  `json:"pathway"`
	VisitSubmittedEpochs []int64 `json:"visit_submitted_epochs"`
}

type Tag struct {
	ID     int64  `json:"id,string"`
	Text   string `json:"text"`
	Common bool   `json:"common"`
}

type TagSavedSearch struct {
	ID           int64  `json:"id,string"`
	Title        string `json:"title"`
	Query        string `json:"query"`
	CreatedEpoch int64  `json:"created_epoch"`
}

func TransformTagMembership(m *model.TagMembership) *TagMembership {
	membership := &TagMembership{
		TagID:   m.TagID,
		CaseID:  m.CaseID,
		Created: m.Created.Unix(),
		Hidden:  m.Hidden,
	}
	if m.TriggerTime != nil {
		triggerTime := m.TriggerTime.Unix()
		membership.TriggerTime = &triggerTime
	}
	return membership
}

func TransformTagSavedSearch(s *model.TagSavedSearch) *TagSavedSearch {
	return &TagSavedSearch{
		ID:           s.ID,
		Title:        s.Title,
		Query:        s.Query,
		CreatedEpoch: s.CreatedTime.Unix(),
	}
}
