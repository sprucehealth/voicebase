package patient

import (
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type CareTeamAssingmentEvent struct {
	PatientID   common.PatientID
	Assignments []*common.CareProviderAssignment
}

type SignupEvent struct {
	AccountID     int64
	PatientID     common.PatientID
	SpruceHeaders *apiservice.SpruceHeaders
}

type AccountLoggedOutEvent struct {
	AccountID int64
}

// VisitStartedEvent is fired when a patient starts their visit.
type VisitStartedEvent struct {
	PatientID     common.PatientID
	VisitID       int64
	PatientCaseID int64
}

// Events implements analytics.Eventer to provide logging of the "visit_started" server event.
func (e *VisitStartedEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:     "visit_started",
			Timestamp: analytics.Time(time.Now()),
			PatientID: e.PatientID.Int64(),
			VisitID:   e.VisitID,
			CaseID:    e.PatientCaseID,
		},
	}
}

// VisitSubmittedEvent is fired when a patient submits their visit.
type VisitSubmittedEvent struct {
	PatientID     common.PatientID
	AccountID     int64
	VisitID       int64
	PatientCaseID int64
	Visit         *common.PatientVisit
	CardID        int64
}

// Events implements analytics.Eventer to provide logging of the "visit_submitted" server event.
func (e *VisitSubmittedEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:     "visit_submitted",
			Timestamp: analytics.Time(time.Now()),
			PatientID: e.PatientID.Int64(),
			VisitID:   e.VisitID,
			CaseID:    e.PatientCaseID,
		},
	}
}

// ParentalConsentCompletedEvent is fired when the parent has completed the consent flow
// and the child's visit is elligible for submission.
type ParentalConsentCompletedEvent struct {
	ChildPatientID  common.PatientID
	ParentPatientID common.PatientID
}

// Events implements analytics.Eventer to provide logging of the "parental_consent_completed" server event.
func (e *ParentalConsentCompletedEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:     "parental_consent_completed",
			Timestamp: analytics.Time(time.Now()),
			PatientID: e.ChildPatientID.Int64(),
			ExtraJSON: analytics.JSONString(struct {
				ParentPatientID int64 `json:"parent_patient_id"`
			}{
				ParentPatientID: e.ParentPatientID.Int64(),
			}),
		},
	}
}
