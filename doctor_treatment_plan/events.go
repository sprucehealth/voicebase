package doctor_treatment_plan

import (
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/common"
)

type NewTreatmentPlanStartedEvent struct {
	PatientID       int64
	DoctorID        int64
	Case            *common.PatientCase
	CaseID          int64
	VisitID         int64
	TreatmentPlanID int64
}

func (e *NewTreatmentPlanStartedEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:           "treatment_plan_started",
			Timestamp:       analytics.Time(time.Now()),
			PatientID:       e.PatientID,
			DoctorID:        e.DoctorID,
			VisitID:         e.VisitID,
			CaseID:          e.CaseID,
			TreatmentPlanID: e.TreatmentPlanID,
		},
	}
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

func (e *TreatmentPlanActivatedEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:           "treatment_plan_activated",
			Timestamp:       analytics.Time(time.Now()),
			PatientID:       e.PatientID,
			DoctorID:        e.DoctorID,
			VisitID:         e.VisitID,
			CaseID:          e.TreatmentPlan.PatientCaseID.Int64(),
			TreatmentPlanID: e.TreatmentPlan.ID.Int64(),
		},
	}
}

type TreatmentPlanSubmittedEvent struct {
	VisitID       int64
	TreatmentPlan *common.TreatmentPlan
}

func (e *TreatmentPlanSubmittedEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:           "treatment_plan_submitted",
			Timestamp:       analytics.Time(time.Now()),
			PatientID:       e.TreatmentPlan.PatientID,
			DoctorID:        e.TreatmentPlan.DoctorID.Int64(),
			VisitID:         e.VisitID,
			CaseID:          e.TreatmentPlan.PatientCaseID.Int64(),
			TreatmentPlanID: e.TreatmentPlan.ID.Int64(),
		},
	}
}
