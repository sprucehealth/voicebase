package doctor_treatment_plan

import "github.com/sprucehealth/backend/common"

type NewTreatmentPlanStartedEvent struct {
	PatientID       int64
	DoctorID        int64
	CaseID          int64
	VisitID         int64
	TreatmentPlanID int64
}

type TreatmentPlanUpdatedEvent struct {
	SectionUpdated  Sections
	DoctorID        int64
	TreatmentPlanID int64
}

type TreatmentPlanActivatedEvent struct {
	PatientID     int64
	DoctorID      int64
	VisitID       int64
	TreatmentPlan *common.TreatmentPlan
	Patient       *common.Patient // Setting Patient is an optional optimization. If this is nil then PatientId can be used.
	Message       *common.CaseMessage
}

type TreatmentPlanSubmittedEvent struct {
	VisitID       int64
	TreatmentPlan *common.TreatmentPlan
}
