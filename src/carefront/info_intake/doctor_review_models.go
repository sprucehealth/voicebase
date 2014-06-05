package info_intake

// Step 1: DOCTOR VISIT REVIEW
type DoctorVisitReviewLayout map[string]interface{}

func (d *DoctorVisitReviewLayout) Get(key string) interface{} {
	m := *d
	return m[key]
}

func NewDoctorVisitReviewLayout() *DoctorVisitReviewLayout {
	dLayout := DoctorVisitReviewLayout(map[string]interface{}{})
	return &dLayout
}

// Step 2: DIAGNOSIS INTAKE
type DiagnosisIntake struct {
	PatientVisitId   int64             `json:"patient_visit_id,string,omitempty"`
	TreatmentPlanId  int64             `json:"treatment_plan_id,string,omitempty"`
	InfoIntakeLayout *InfoIntakeLayout `json:"health_condition"`
}
