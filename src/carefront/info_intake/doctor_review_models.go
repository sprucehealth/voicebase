package info_intake

import "carefront/api"

// Step 1: DOCTOR VISIT REVIEW
type DoctorVisitReviewLayout map[string]interface{}

func (d DoctorVisitReviewLayout) FillInDatabaseInfo(dataApi api.DataAPI, languageId int64) error {
	// Nothing to do here given that we are only filling in information directly based on what the patient answers
	// at the time of presenting the information to the doctor
	return nil
}

func (d DoctorVisitReviewLayout) GetHealthConditionTag() string {
	return d["health_condition"].(string)
}

// Step 2: DIAGNOSIS INTAKE
type DiagnosisIntake struct {
	PatientVisitId   int64             `json:"patient_visit_id,string,omitempty"`
	TreatmentPlanId  int64             `json:"treatment_plan_id,string,omitempty"`
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
	switch purpose {
	case api.DIAGNOSE_PURPOSE:
		return &DiagnosisIntake{}
	case api.REVIEW_PURPOSE:
		d := DoctorVisitReviewLayout(map[string]interface{}{})
		return &d
	}

	return nil
}
