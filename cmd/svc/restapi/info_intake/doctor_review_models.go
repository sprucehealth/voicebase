package info_intake

// Step 2: DIAGNOSIS INTAKE
type DiagnosisIntake struct {
	PatientVisitID   int64             `json:"patient_visit_id,string,omitempty"`
	InfoIntakeLayout *InfoIntakeLayout `json:"health_condition"`
}