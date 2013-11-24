package common

import (
	"time"
)

type Patient struct {
	PatientId int64     `json:"id,omitempty,string"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omiempty"`
	Dob       time.Time `json:"dob,omitempty"`
	Gender    string    `json:"gender,omitempty"`
	Zipcode   string    `json:"zip_code,omitempty"`
	Status    string    `json:"-"`
	AccountId int64     `json:"-"`
}

type PatientVisit struct {
	PatientVisitId    int64     `json:"patient_visit_id,string,omitempty"`
	PatientId         int64     `json:"patient_id,string,omitempty"`
	CreationDate      time.Time `json:"creation_date,omitempty`
	OpenedDate        time.Time `json:"opened_date,omitempty"`
	ClosedDate        time.Time `json:"closed_date,omitempty"`
	HealthConditionId int64     `json:"health_condition_id,omitempty,string"`
	Status            string    `json:"status,omitempty"`
	LayoutVersionId   int64     `json:"layout_version_id,omitempty,string"`
}

type PatientAnswer struct {
	PatientAnswerId   int64            `json:"patient_answer_id,string,omitempty"`
	QuestionId        int64            `json:"-"`
	PatientId         int64            `json:"-"`
	PatientVisitId    int64            `json:"-"`
	ParentQuestionId  int64            `json:"-"`
	ParentAnswerId    int64            `json:"-"`
	PotentialAnswerId int64            `json:"potential_answer_id,string,omitempty"`
	PotentialAnswer   string           `json:"potential_answer,omitempty"`
	AnswerSummary     string           `json:"answer_summary,omitempty"`
	LayoutVersionId   int64            `json:"-"`
	SubAnswers        []*PatientAnswer `json:"answers,omitempty"`
	AnswerText        string           `json:"answer_text,omitempty"`
	ObjectUrl         string           `json:"object_url,omitempty"`
	StorageBucket     string           `json:"-"`
	StorageKey        string           `json:"-"`
	StorageRegion     string           `json:"-"`
}
