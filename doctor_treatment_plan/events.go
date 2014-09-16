package doctor_treatment_plan

import "github.com/sprucehealth/backend/common"

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

type TreatmentPlanActivatedEvent struct {
	PatientId     int64
	DoctorId      int64
	VisitId       int64
	TreatmentPlan *common.DoctorTreatmentPlan
	Patient       *common.Patient // Setting Patient is an optional optimization. If this is nil then PatientId can be used.
	Message       *common.CaseMessage
}

type TreatmentPlanSubmittedEvent struct {
	VisitId       int64
	TreatmentPlan *common.DoctorTreatmentPlan
}
