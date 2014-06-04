package common

import (
	"carefront/encoding"
	"time"
)

type Answer interface {
	getQuestionId() int64
}

type AnswerIntake struct {
	AnswerIntakeId    encoding.ObjectId `json:"answer_id,omitempty"`
	QuestionId        encoding.ObjectId `json:"-"`
	RoleId            encoding.ObjectId `json:"-"`
	Role              string            `json:"-"`
	ContextId         encoding.ObjectId `json:"-"`
	ParentQuestionId  encoding.ObjectId `json:"-"`
	ParentAnswerId    encoding.ObjectId `json:"-"`
	PotentialAnswerId encoding.ObjectId `json:"potential_answer_id,omitempty"`
	PotentialAnswer   string            `json:"potential_answer,omitempty"`
	AnswerSummary     string            `json:"potential_answer_summary,omitempty"`
	LayoutVersionId   encoding.ObjectId `json:"-"`
	SubAnswers        []*AnswerIntake   `json:"answers,omitempty"`
	AnswerText        string            `json:"answer_text,omitempty"`
	ObjectUrl         string            `json:"object_url,omitempty"`
	StorageBucket     string            `json:"-"`
	StorageKey        string            `json:"-"`
	StorageRegion     string            `json:"-"`
	ToAlert           bool              `json:"-"`
}

func (a *AnswerIntake) getQuestionId() int64 {
	return a.QuestionId.Int64()
}

type PhotoIntakeSection struct {
	Id           int64              `json:"-"`
	QuestionId   int64              `json:"-"`
	Name         string             `json:"name,omitempty"`
	Photos       []*PhotoIntakeSlot `json:"photos,omitempty"`
	CreationDate time.Time          `json:"creation_date"`
}

func (p *PhotoIntakeSection) getQuestionId() int64 {
	return p.QuestionId
}

type PhotoIntakeSlot struct {
	Id           int64     `json:"-"`
	CreationDate time.Time `json:"creation_date"`
	PhotoUrl     string    `json:"photo_url"`
	PhotoId      int64     `json:"photo_id,string,omitempty"`
	SlotId       int64     `json:"slot_id,string"`
	Name         string    `json:"name"`
}
