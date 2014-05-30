package visit

type DiagnosisModifiedEvent struct {
	DoctorId        int64
	PatientVisitId  int64
	TreatmentPlanId int64
}

type PatientVisitMarkedUnsuitableEvent struct {
	PatientVisitId int64
	DoctorId       int64
}
