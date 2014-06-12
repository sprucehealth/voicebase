package doctor_treatment_plan

import "carefront/common"

type NewTreatmentPlanStartedEvent struct {
	DoctorId        int64
	PatientVisitId  int64
	TreatmentPlanId int64
}

type TreatmentsAddedEvent struct {
	TreatmentPlanId int64
	DoctorId        int64
	Treatments      []*common.Treatment
}

type RegimenPlanAddedEvent struct {
	DoctorId        int64
	TreatmentPlanId int64
	RegimenPlan     *common.RegimenPlan
}

type AdviceAddedEvent struct {
	DoctorId        int64
	TreatmentPlanId int64
	Advice          *common.Advice
}
