package patient_visit

type VisitChargedEvent struct {
	PatientID     int64
	VisitID       int64
	PatientCaseID int64
}

type DiagnosisModifiedEvent struct {
	PatientID       int64
	DoctorID        int64
	PatientVisitID  int64
	TreatmentPlanID int64
	Diagnosis       string
	PatientCaseID   int64
}

type PatientVisitMarkedUnsuitableEvent struct {
	PatientVisitID int64
	PatientID      int64
	CaseID         int64
	DoctorID       int64
	InternalReason string
}
