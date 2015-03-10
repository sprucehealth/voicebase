package patient_visit

type DiagnosisModifiedEvent struct {
	PatientID       int64
	DoctorID        int64
	PatientVisitID  int64
	TreatmentPlanID int64
	PatientCaseID   int64
}

type PatientVisitMarkedUnsuitableEvent struct {
	PatientVisitID int64
	PatientID      int64
	CaseID         int64
	DoctorID       int64
	InternalReason string
}

type PreSubmissionVisitTriageEvent struct {
	VisitID       int64
	CaseID        int64
	Title         string
	ActionMessage string
	ActionURL     string
}
