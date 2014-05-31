package doctor_treatment_plan

type NewTreatmentPlanStartedEvent struct {
	DoctorId        int64
	PatientVisitId  int64
	TreatmentPlanId int64
}
