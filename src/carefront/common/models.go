package common

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
