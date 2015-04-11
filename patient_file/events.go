package patient_file

import (
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/common"
)

type PatientVisitOpenedEvent struct {
	PatientVisit *common.PatientVisit
	PatientID    int64
	DoctorID     int64
	Role         string
}

func (e *PatientVisitOpenedEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:     "visit_opened",
			Timestamp: analytics.Time(time.Now()),
			PatientID: e.PatientID,
			DoctorID:  e.DoctorID,
			VisitID:   e.PatientVisit.PatientVisitID.Int64(),
			CaseID:    e.PatientVisit.PatientCaseID.Int64(),
			Role:      e.Role,
		},
	}
}
