package patient_visit

type VisitStartedEvent struct {
	PatientId int64
	VisitId   int64
}

type VisitSubmittedEvent struct {
	PatientId int64
	DoctorId  int64
	VisitId   int64
}
