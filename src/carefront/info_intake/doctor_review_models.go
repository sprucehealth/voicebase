package info_intake

import (
	"carefront/api"
	"carefront/common"
	"reflect"
	"time"
)

// Step 1: PATIENT VISIT REVIEW

type PatientVisitOverviewQuestion struct {
	Question
	ShowPotentialResponses bool   `json:"show_potential_responses,omitempty"`
	FlagQuestionIfAnswered bool   `json:"flag_question_if_answered,omitempty"`
	GenderFilter           string `json:"gender,omitempty"`
}

type PatientVisitOverviewSubSection struct {
	Questions       []*PatientVisitOverviewQuestion `json:"data,omitempty"`
	SubSectionTitle string                          `json:"sub_section_title,omitempty"`
	SubSectionTypes []string                        `json:"sub_section_types,omitempty"`
	GenderFilter    string                          `json:"gender,omitempty"`
}

type PatientVisitOverviewSection struct {
	SectionTitle string                            `json:"section_title,omitempty"`
	SectionTypes []string                          `json:"section_types,omitempty"`
	SubSections  []*PatientVisitOverviewSubSection `json:"sub_sections,omitempty"`
}

type PatientVisitOverview struct {
	PatientVisitTime   time.Time                      `json:"patient_visit_time,omitempty"`
	Patient            *common.Patient                `json:"patient,omitempty"`
	PatientVisitId     int64                          `json:"patient_visit_id,string,omitempty"`
	HealthConditionId  int64                          `json:"health_condition_id,string,omitempty"`
	HealthConditionTag string                         `json:"health_condition,omitempty"`
	Sections           []*PatientVisitOverviewSection `json:"sections,omitempty"`
}

func (p *PatientVisitOverview) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	// fill in the questions from the database
	for _, patientVisitSection := range p.Sections {
		for _, subSection := range patientVisitSection.SubSections {
			for _, question := range subSection.Questions {
				// assume that if the show_potential_responses is not set,
				// it means that potential responses should not be shown
				if !reflect.ValueOf(question.ShowPotentialResponses).IsValid() {
					question.ShowPotentialResponses = false
				}
				question.FillInDatabaseInfo(dataApi, languageId)
			}
		}
	}
	return nil
}

func (p *PatientVisitOverviewQuestion) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	err := p.Question.FillInDatabaseInfo(dataApi, languageId)

	// removing the potential responses if the flag indicates that it should not be present
	if p.ShowPotentialResponses == false {
		p.Question.PotentialAnswers = nil
	}

	return err
}

func (p *PatientVisitOverview) GetHealthConditionTag() string {
	return p.HealthConditionTag
}

// Step 2: DIAGNOSIS INTAKE
type DiagnosisIntake struct {
	PatientVisitId   int64             `json:"patient_visit_id,string,omitempty"`
	InfoIntakeLayout *InfoIntakeLayout `json:"health_condition"`
}

func (d *DiagnosisIntake) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	// fill in the questions from the database
	for _, section := range d.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {
			err := question.FillInDatabaseInfo(dataApi, languageId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *DiagnosisIntake) GetHealthConditionTag() string {
	return d.InfoIntakeLayout.HealthConditionTag
}

func GetLayoutModelBasedOnPurpose(purpose string) InfoIntakeModel {
	if purpose == api.REVIEW_PURPOSE {
		return &PatientVisitOverview{}
	} else if purpose == api.DIAGNOSE_PURPOSE {
		return &DiagnosisIntake{}
	}
	return nil
}
