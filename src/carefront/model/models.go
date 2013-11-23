package model

type PatientAnswer struct {
	PatientAnswerId   int64            `json:"patient_answer_id,string"`
	QuestionId        int64            `json:"question_id,omitempty,string"`
	PatientId         int64            `json:"patient_id",omitempty,string"`
	PatientVisitId    int64            `json:"patient_visit_id,omitempty,string"`
	ParentQuestionId  int64            `json:"parent_question_id,string"`
	ParentAnswerId    int64            `json:"parent_answer_id,string"`
	PotentialAnswerId int64            `json:"potential_answer_id,string"`
	LayoutVersionId   int64            `json:"layout_version_id,string,omitempty"`
	SubAnswers        []*PatientAnswer `json:"answers,omitempty"`
	AnswerText        string           `json:"answer_text,omitempty"`
	ObjectUrl         string           `json:"object_url,omitempty"`
	StorageBucket     string           `json:"storage_bucket,omitempty"`
	StorageKey        string           `json:"storage_key,omitempty"`
	StorageRegion     string           `json:"storage_region,omitempty"`
}
