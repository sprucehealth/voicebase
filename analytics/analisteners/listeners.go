package analisteners

import (
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/patient_visit"
)

func InitListeners(analyticsLogger analytics.Logger) {
	// Doctor treatment plan events

	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.NewTreatmentPlanStartedEvent) error {
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:           "treatment_plan_started",
				Timestamp:       analytics.Time(time.Now()),
				PatientID:       ev.PatientID,
				DoctorID:        ev.DoctorID,
				VisitID:         ev.VisitID,
				CaseID:          ev.CaseID,
				TreatmentPlanID: ev.TreatmentPlanID,
			},
		})
		return nil
	})
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:           "treatment_plan_activated",
				Timestamp:       analytics.Time(time.Now()),
				PatientID:       ev.PatientId,
				DoctorID:        ev.DoctorId,
				VisitID:         ev.VisitId,
				CaseID:          ev.TreatmentPlan.PatientCaseId.Int64(),
				TreatmentPlanID: ev.TreatmentPlan.Id.Int64(),
			},
		})
		return nil
	})
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanSubmittedEvent) error {
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:           "treatment_plan_submitted",
				Timestamp:       analytics.Time(time.Now()),
				PatientID:       ev.TreatmentPlan.PatientId,
				DoctorID:        ev.TreatmentPlan.DoctorId.Int64(),
				VisitID:         ev.VisitId,
				CaseID:          ev.TreatmentPlan.PatientCaseId.Int64(),
				TreatmentPlanID: ev.TreatmentPlan.Id.Int64(),
			},
		})
		return nil
	})

	// Patient events

	dispatch.Default.Subscribe(func(ev *patient.VisitStartedEvent) error {
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "visit_started",
				Timestamp: analytics.Time(time.Now()),
				PatientID: ev.PatientId,
				VisitID:   ev.VisitId,
				CaseID:    ev.PatientCaseId,
			},
		})
		return nil
	})
	dispatch.Default.Subscribe(func(ev *patient.VisitSubmittedEvent) error {
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "visit_submitted",
				Timestamp: analytics.Time(time.Now()),
				PatientID: ev.PatientId,
				VisitID:   ev.VisitId,
				CaseID:    ev.PatientCaseId,
			},
		})
		return nil
	})

	// Patient visit events

	dispatch.Default.Subscribe(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "visit_marked_unsuitable",
				Timestamp: analytics.Time(time.Now()),
				PatientID: ev.PatientID,
				DoctorID:  ev.DoctorID,
				VisitID:   ev.PatientVisitID,
				CaseID:    ev.CaseID,
			},
		})
		return nil
	})
	dispatch.Default.Subscribe(func(ev *patient_visit.VisitChargedEvent) error {
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "visit_charged",
				Timestamp: analytics.Time(time.Now()),
				PatientID: ev.PatientID,
				VisitID:   ev.VisitID,
				CaseID:    ev.PatientCaseID,
			},
		})
		return nil
	})
	dispatch.Default.Subscribe(func(ev *patient_visit.DiagnosisModifiedEvent) error {
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:           "diagnosis_modified",
				Timestamp:       analytics.Time(time.Now()),
				PatientID:       ev.PatientID,
				DoctorID:        ev.DoctorID,
				VisitID:         ev.PatientVisitID,
				CaseID:          ev.PatientCaseID,
				TreatmentPlanID: ev.TreatmentPlanID,
			},
		})
		return nil
	})

	// Patient file events

	dispatch.Default.Subscribe(func(ev *patient_file.PatientVisitOpenedEvent) error {
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "visit_opened",
				Timestamp: analytics.Time(time.Now()),
				PatientID: ev.PatientId,
				DoctorID:  ev.DoctorId,
				VisitID:   ev.PatientVisit.PatientVisitId.Int64(),
				CaseID:    ev.PatientVisit.PatientCaseId.Int64(),
				Role:      ev.Role,
			},
		})
		return nil
	})
}
