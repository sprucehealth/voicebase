package patient

import (
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type CareTeamAssingmentEvent struct {
	PatientID   int64
	Assignments []*common.CareProviderAssignment
}

type SignupEvent struct {
	AccountID     int64
	PatientID     int64
	SpruceHeaders *apiservice.SpruceHeaders
}

type AccountLoggedOutEvent struct {
	AccountID int64
}

type VisitStartedEvent struct {
	PatientID     int64
	VisitID       int64
	PatientCaseID int64
}

func (e *VisitStartedEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:     "visit_started",
			Timestamp: analytics.Time(time.Now()),
			PatientID: e.PatientID,
			VisitID:   e.VisitID,
			CaseID:    e.PatientCaseID,
		},
	}
}

type VisitSubmittedEvent struct {
	PatientID     int64
	AccountID     int64
	VisitID       int64
	PatientCaseID int64
	Visit         *common.PatientVisit
	CardID        int64
}

func (e *VisitSubmittedEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:     "visit_submitted",
			Timestamp: analytics.Time(time.Now()),
			PatientID: e.PatientID,
			VisitID:   e.VisitID,
			CaseID:    e.PatientCaseID,
		},
	}
}
