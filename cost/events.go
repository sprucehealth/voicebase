package cost

import (
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/common"
)

type VisitChargedEvent struct {
	AccountID     int64
	PatientID     common.PatientID
	VisitID       int64
	IsFollowup    bool
	PatientCaseID int64
}

func (e *VisitChargedEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:     "visit_charged",
			Timestamp: analytics.Time(time.Now()),
			PatientID: e.PatientID.Int64(),
			VisitID:   e.VisitID,
			CaseID:    e.PatientCaseID,
		},
	}
}
