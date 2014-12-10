package doctor_treatment_plan

import "github.com/sprucehealth/backend/common"

type NewTreatmentPlanStartedEvent struct {
	PatientID       int64
	DoctorID        int64
	CaseID          int64
	VisitID         int64
	TreatmentPlanID int64
}

type TreatmentsAddedEvent struct {
	TreatmentPlanID int64
	DoctorID        int64
	Treatments      []*common.Treatment
}

type RegimenPlanAddedEvent struct {
	DoctorID        int64
	TreatmentPlanID int64
	RegimenPlan     *common.RegimenPlan
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

type TreatmentPlanNoteUpdatedEvent struct {
	DoctorID        int64
	TreatmentPlanID int64
	Note            string
}

type TreatmentPlanScheduledMessagesUpdatedEvent struct {
	DoctorID        int64
	TreatmentPlanID int64
}
