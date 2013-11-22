package info_intake

import (
	"time"
)

type PatientVisitOverviewQuestion struct {
	Question
	ShowPotentialResponses bool `json:"show_potential_responses"`
}

type PatientVisitOverviewSubSection struct {
	Questions       []*PatientVisitOverviewQuestion `json:"data,omitempty"`
	SubSectionTitle string                          `json:"sub_section_title,omitempty"`
	SubSectionTypes []string                        `json:"sub_section_types,omitempty"`
}

type PatientVisitOverviewSection struct {
	SectionTitle string                            `json:"section_title,omitempty"`
	SectionTypes []string                          `json:"section_types,omitempty"`
	SubSections  []*PatientVisitOverviewSubSection `json:"sub_sections,omitempty"`
}

type PatientVisitOverview struct {
	PatientVisitTime  time.Time                      `json:"patient_visit_time,empty"`
	PatientId         int64                          `json:"patient_id,string,omitempty"`
	PatientVisitId    int64                          `json:"patient_visit_id,string,omitempty"`
	HealthConditionId int64                          `json:"health_condition_id,string,omitempty"`
	Sections          []*PatientVisitOverviewSection `json:"sections,omitempty"`
}
